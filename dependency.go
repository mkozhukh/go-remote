package go_remote

import (
	"context"
	"errors"
	"reflect"
)

type dataRecord struct {
	isConstant bool
	rtype      reflect.Type
	value      interface{}
}

type DependencyProvider func(ctx context.Context) interface{}

type dependencyStore struct {
	data map[reflect.Type]reflect.Value
}

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()
var contextInterface = reflect.TypeOf((*context.Context)(nil)).Elem()

func newDependencyStore() *dependencyStore {
	return &dependencyStore{data: make(map[reflect.Type]reflect.Value)}
}

func (d *dependencyStore) AddProvider(provider interface{}) error {
	pType := reflect.TypeOf(provider)
	if pType.NumIn() != 1 || !pType.In(0).Implements(contextInterface) {
		msg := "invalid data provider, provider must have a context as incoming parameter"
		log.Error(msg)
		return errors.New(msg)
	}
	if pType.NumOut() != 1 && (pType.NumOut() != 2 || pType.Out(1).Implements(errorInterface)) {
		msg := "invalid data provider, provider must return a value and optional error"
		log.Error(msg)
		return errors.New(msg)
	}

	retType := pType.Out(0)
	if retType.Kind() == reflect.Ptr {
		retType = retType.Elem()
	}
	d.data[retType] = reflect.ValueOf(provider)
	return nil
}

func (d *dependencyStore) Value(rtype reflect.Type, ctx context.Context) (reflect.Value, bool, error) {
	keyType := rtype
	if rtype.Kind() == reflect.Ptr {
		keyType = rtype.Elem()
	}

	test, ok := d.data[keyType]
	if !ok {
		return reflect.Value{}, false, nil
	}

	var args []reflect.Value
	args = []reflect.Value{reflect.ValueOf(ctx)}

	out := test.Call(args)
	if len(out) > 1 {
		err, _ := out[1].Interface().(error)
		if err != nil {
			log.Error("error during calculation", "name", rtype.Name(), "err", err.Error())
		}
		return out[0], true, err
	}

	return out[0], true, nil
}
