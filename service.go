package remote

import (
	"encoding/json"
	"errors"
	"reflect"
	"unicode"
	"unicode/utf8"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type serviceProvider struct {
	*provider

	methods  map[string]*providerInfo
	services map[string]*serviceProvider
}

func newServiceProvider(di *provider) *serviceProvider {
	t := serviceProvider{}
	t.provider = di

	t.methods = make(map[string]*providerInfo)
	t.services = make(map[string]*serviceProvider)

	return &t
}

func (s *serviceProvider) Add(name string, source interface{}, guard interface{}) {

	if reflect.ValueOf(source).Kind() == reflect.Func {
		vtype := reflect.TypeOf(source)
		value := reflect.ValueOf(source)
		s.methods[name] = s.getProviderInfo(vtype, value, guard)
	} else {
		value := reflect.ValueOf(source)
		if value.Kind() != reflect.Ptr {
			if value.Kind() != reflect.Struct {
				log.Fatalf("Not supported service type for %s, %+v", name, source)
			}
		} else {
			value := reflect.Indirect(value)
			if value.Kind() != reflect.Struct {
				log.Fatalf("Not supported service type for %s, %+v", name, source)
			}
		}

		t := newServiceProvider(s.provider)

		vtype := value.Type()
		if name == "" {
			name = vtype.Name()
		}

		t.suitableMethods(vtype, &value, guard)
		s.services[name] = t
	}
}

// ToJSON converts object to JSON struct
func (s *serviceProvider) ToJSON() ([]byte, error) {
	all := make(map[string][]byte)
	for key := range s.methods {
		all[key] = []byte{0x31}
	}

	for key := range s.services {
		all[key], _ = s.services[key].ToJSON()
	}

	return json.Marshal(all)
}

func (s *serviceProvider) Value(thecall *callInfo) (data interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("panic occurs")
		}
	}()

	log.Debugf("%s %+v", thecall.Name, s.methods)

	afunc, ok := s.methods[thecall.Name]
	if ok {
		return s.callAndResolve(afunc, thecall)
	}

	objname := thecall.SplitName()
	aobj, ok := s.services[objname]
	if ok {
		return aobj.Value(thecall)
	}

	return reflect.Value{}, errors.New("unknown method")
}

func (s *serviceProvider) callAndResolve(method *providerInfo, thecall callState) (interface{}, error) {
	var out interface{}

	data, err := s.call(method, thecall)
	if err == nil {
		count := method.vtype.NumOut()
		for i := 0; i < count; i++ {
			if method.vtype.Out(i) == typeOfError {
				errResult := data[i].Interface()
				if errResult != nil {
					err = errResult.(error)
				}
			} else {
				out = data[i].Interface()
			}
		}
	}

	return out, err
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// check all methods on an object and return public ones
func (s serviceProvider) suitableMethods(typ reflect.Type, value *reflect.Value, guard interface{}) {
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name

		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}

		in := mtype.NumIn()
		out := mtype.NumOut()
		inTypes := make([]reflect.Type, in)
		outTypes := make([]reflect.Type, out)

		for i := 0; i < in; i++ {
			argType := mtype.In(i)
			inTypes[i] = argType
			if !isExportedOrBuiltinType(argType) {
				log.Fatalf("[remote] argument type not exported for method %s", mname)
			}
		}
		for i := 0; i < out; i++ {
			argType := mtype.Out(i)
			outTypes[i] = argType
			if !isExportedOrBuiltinType(argType) {
				log.Fatalf("[remote] result type not exported for method %s", mname)
			}
		}

		info := s.getProviderInfo(method.Type, method.Func, guard)
		info.object = value
		s.methods[mname] = info
	}
}
