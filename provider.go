package remote

import (
	"errors"
	"net/http"
	"reflect"
)

type dataRecord struct {
	isConstant bool
	rtype      reflect.Type
	value      interface{}
}

func newDataProvider() dataProvider {
	t := dataProvider{}
	t.data = make(map[reflect.Type]reflect.Value)
	return t
}

type dataProvider struct {
	data map[reflect.Type]reflect.Value
}

func (d *dataProvider) Add(provider interface{}) error {
	pType := reflect.TypeOf(provider)
	if pType.Kind() != reflect.Func || pType.NumOut() != 1 || pType.NumIn() != 1 || pType.In(0) != requestType {
		msg := "Invalid parameter for RegisterProvider, function factory is expected"
		log.Errorf(msg)
		return errors.New(msg)
	}

	d.data[pType.Out(0)] = reflect.ValueOf(provider)
	return nil
}

func (d *dataProvider) Value(rtype *reflect.Type, req *http.Request) (reflect.Value, error) {
	test, ok := d.data[*rtype]
	if !ok {
		return reflect.Value{}, errors.New("Missed parameter in method call")
	}

	args := []reflect.Value{reflect.ValueOf(req)}
	return test.Call(args)[0], nil
}
