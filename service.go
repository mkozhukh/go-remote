package go_remote

import (
	"reflect"
	"runtime/debug"
	"unicode"
	"unicode/utf8"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type methodType struct {
	method   reflect.Method
	inTypes  []reflect.Type
	outTypes []reflect.Type
}

type service struct {
	name   string        // name of service
	rcvr   reflect.Value // receiver of methods for the service
	typ    reflect.Type  // type of the receiver
	guard  Guard
	method map[string]*methodType // registered methods
}

func valueByType(atype reflect.Type, i int, thecall *callInfo, args callArgs) (reflect.Value, error) {
	var argv reflect.Value

	val, ok, err := thecall.dependencies.Value(atype, thecall.ctx)
	if ok {
		return val, err
	}

	// Decode the argument value
	argIsValue := false // if true, need to indirect before calling.
	if atype.Kind() == reflect.Ptr {
		argv = reflect.New(atype.Elem())
	} else {
		argv = reflect.New(atype)
		argIsValue = true
	}

	// argv guaranteed to be a pointer now.
	if err := args.ReadArgument(i, argv.Interface()); err != nil {
		return argv, err
	}
	if argIsValue {
		argv = argv.Elem()
	}

	return argv, nil
}

func (s *service) Call(thecall *callInfo, args callArgs, res *Response) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("method call error", "error", r)
			log.Error(string(debug.Stack()))
			res.Error = "Method call error"
		}
	}()

	var err error

	if s.guard != nil && !s.guard(thecall.ctx) {
		res.Error = "Access Denied"
		return
	}

	mtype, ok := s.method[thecall.method]
	if !ok {
		res.Error = "Invalid method name"
		log.Warn(res.Error)
		return
	}

	argv := make([]reflect.Value, len(mtype.inTypes))
	argv[0] = s.rcvr
	for i := 1; i < len(mtype.inTypes); i++ {
		argv[i], err = valueByType(mtype.inTypes[i], i-1, thecall, args)
		if err != nil {
			res.Error = err.Error()
			log.Warn("Invalid arguments", "arg", err)
			return
		}
	}

	dv := make([]interface{}, len(argv))
	for i := 1; i < len(argv); i++ {
		dv[i] = argv[i].Interface()
	}
	log.Debug("args", "msg", dv)

	// Invoke the method
	returnValues := mtype.method.Func.Call(argv)

	var outResult interface{}
	for i := 0; i < len(mtype.outTypes); i++ {
		if mtype.outTypes[i] == typeOfError {
			errResult := returnValues[i].Interface()
			if errResult != nil {
				res.Error = errResult.(error).Error()
				return
			}
		} else {
			outResult = returnValues[i].Interface()
		}
	}

	res.Data = outResult
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

// creates a new service object
func newService(rcvr interface{}, guard Guard) *service {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	s.guard = guard

	// install the methods
	s.method = suitableMethods(s.typ, true)

	return s
}

// check all methods on an object and return public ones
func suitableMethods(typ reflect.Type, reportErr bool) map[string]*methodType {
	methods := make(map[string]*methodType)
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
				log.Error(mname, "argument type not exported", "type", argType)
			}
		}
		for i := 0; i < out; i++ {
			argType := mtype.Out(i)
			outTypes[i] = argType
			if !isExportedOrBuiltinType(argType) {
				log.Error(mname, "result type not exported", "type", argType)
			}
		}

		methods[mname] = &methodType{method, inTypes, outTypes}
	}
	return methods
}
