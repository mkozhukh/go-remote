package go_remote

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
)

// Guard is a guard function that allows or denies code execution based on the context
type Guard = func(r context.Context) bool

// Connect extends of blocks request based on context value
type Connect = func(r context.Context) (context.Context,error)

// Server structure stores all methods, events and data of API
type Server struct {
	services map[string]*service
	data     map[string]dataRecord
	config   *ServerConfig

	Connect 	 Connect
	Events       *Hub
	Dependencies *dependencyStore
}

type ServerConfig struct {
	WebSocket  bool
	WithoutKey bool
}

// Response handles results of remote calls
type Response struct {
	ID    string      `json:"id"`
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
}

// NewServer creates a new Server instance
func NewServer(config *ServerConfig) *Server {
	s := Server{}
	s.services = make(map[string]*service)
	s.data = make(map[string]dataRecord)
	s.config = config

	if s.config == nil {
		s.config = &ServerConfig{}
	}

	s.Events = newHub()

	if s.config.WebSocket {
		go s.Events.Run()
	}

	s.Dependencies = newDependencyStore()
	s.Connect = func(ctx context.Context) (context.Context,error) { return ctx, nil }
	return &s
}

// AddService exposes all public methods of the provided object
func (s *Server) AddService(name string, rcvr interface{}) error {
	return s.register(name, rcvr, nil)
}

// AddServiceWithGuard exposes all public methods of the provided object with a guard
func (s *Server) AddServiceWithGuard(name string, rcvr interface{}, guard Guard) error {
	return s.register(name, rcvr, guard)
}

// AddVariable adds a variable data to the API
func (s *Server) AddVariable(name string, rcvr interface{}) error {
	return s.registerData(name, rcvr, false)
}

// AddConstant adds a constant data to the API
func (s *Server) AddConstant(name string, rcvr interface{}) error {
	return s.registerData(name, rcvr, true)
}

func (s *Server) registerData(name string, rcvr interface{}, isConstant bool) error {
	if _, ok := s.data[name]; ok {
		return errors.New("service name already used")
	}

	if isConstant {
		if reflect.TypeOf(rcvr).Kind() == reflect.Ptr {
			rcvr = reflect.ValueOf(rcvr).Elem().Interface()
		}
		s.data[name] = dataRecord{isConstant: true, value: rcvr}
	} else {
		s.data[name] = dataRecord{isConstant: false, rtype: reflect.TypeOf(rcvr)}
	}

	return nil
}

func (s *Server) register(name string, rcvr interface{}, guard Guard) error {
	service := newService(rcvr, guard)
	if name == "" {
		name = service.name
	}
	// store the service
	s.services[name] = service
	return nil
}

// Process starts the package processing, executing all requested methods
func (s *Server) Process(input []byte, c context.Context) []Response {
	data := callData{}
	err := json.Unmarshal(input, &data)
	response := make([]Response, len(data))

	if err != nil {
		log.Errorf(err.Error())
		return response
	}

	res := make(chan *Response)

	for i := range data {
		data[i].parse()
		data[i].dependencies = s.Dependencies
		data[i].ctx = c

		go s.Call(data[i], res)
	}

	for i := range data {
		response[i] = *(<-res)
	}

	return response
}

// Call allows to execute some Servers's method
func (s *Server) Call(call *callInfo, res chan *Response) {
	response := Response{ID: call.ID}

	log.Debugf("Call %s.%s", call.service, call.method)
	service, ok := s.services[call.service]
	if !ok {
		response.Error = "Unknown service"
	} else {
		service.Call(call, &response)
	}

	res <- &response
}
