package remote

import (
	"errors"
	"net/http"
	"reflect"
)

var requestType = reflect.TypeOf(&http.Request{})

type providerState struct {
	request *http.Request
	response http.ResponseWriter
}

type providerInfo struct {
	vtype   reflect.Type
	value	reflect.Value
	guard	*reflect.Value
	object  *reflect.Value
}

type provider struct {
	data map[reflect.Type]*providerInfo
}

func newProvider() *provider {
	t := provider{}
	t.data = make(map[reflect.Type]*providerInfo)
	return &t
}

func (d *provider) Add(provider interface{}, guard interface{}) error {
	pType := reflect.TypeOf(provider)
	pValue := reflect.ValueOf(provider)

	info := d.getProviderInfo(pType, pValue, guard)

	d.data[info.vtype.Out(0)] = info
	return nil
}

func (d *provider) getProviderInfo(pType reflect.Type, value reflect.Value, guard interface{}) *providerInfo {
	if pType.Kind() != reflect.Func || pType.NumOut() == 0 {
		log.Fatalf("invalid dependency provider")
	}

	if guard != nil {
		gType := reflect.TypeOf(guard)

		if gType.NumOut() != 1  || pType.Out(0).Kind() != reflect.Bool {
			log.Fatalf("invalid guard")
		}

		guardValue := reflect.ValueOf(guard)
		return &providerInfo{pType,value, &guardValue, nil }
	} else {
		return &providerInfo{pType,value, nil, nil}
	}
}

func (d *provider) Value(rtype *reflect.Type, state callState) (reflect.Value, error) {
	test, ok := d.data[*rtype]
	if !ok {
		return reflect.Value{}, errors.New("can't resolve dependency")
	}

	data, err := d.call(test, state)
	if data == nil || len(data) == 0 {
		return reflect.Value{}, err
	}

	return data[0], err
}

func (s *provider) call(method *providerInfo, thecall callState) ([]reflect.Value, error) {
	if method.guard != nil && !s.checkGuard(method.guard, thecall) {
		return nil, errors.New("access denied")
	}

	data, err := s.resolve(&method.value, thecall)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (d *provider) checkGuard(rtype *reflect.Value, state callState) bool {
	data, err := d.resolve(rtype, state)
	if err != nil || !data[0].Bool() {
		return false
	}

	return true
}

func (d *provider) resolve(rtype *reflect.Value, state callState) ([]reflect.Value, error) {
	signature := rtype.Type()
	count := signature.NumIn()

	args := make([]reflect.Value, 0, count)
	for i := range args {
		intype := signature.In(i)

		//request
		if intype == requestType {
			args = append(args, reflect.ValueOf(state.Request()))
		}

		//from provider
		arg, ok := d.data[intype]
		if ok {
			val, err := d.call(arg, &innerCallInfo{state.Request()})
			if err != nil{
				return nil, errors.New("can't resolve dependency")
			}
			args = append(args, val[0])
		}

		//from the call
		var argValue reflect.Value
		argIsValue := false // if true, need to indirect before calling.
		if intype.Kind() == reflect.Ptr {
			argValue = reflect.New(intype.Elem())
		} else {
			argValue = reflect.New(intype)
			argIsValue = true
		}

		err := state.NextArgument(argValue.Interface())
		if err != nil {
			return nil, err
		}

		if (argIsValue){
			args = append(args, argValue.Elem())
		} else {
			args = append(args, argValue)
		}
	}

	return rtype.Call(args), nil
}