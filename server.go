package remote

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
)

var requestType = reflect.TypeOf(&http.Request{})

// Server structure stores all methods and data of API
type Server struct {
	Version      int
	CookieName   string
	services     map[string]*service
	data         map[string]dataRecord
	dependencies dataProvider
}

var log Logger = defaultLogger{}

// Logger object, logrus interface is used by default
type Logger interface {
	Errorf(string, ...interface{})
	Debugf(string, ...interface{})
}

type defaultLogger struct{}

func (l defaultLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("ERROR: "+format+"\n", args...)
}

func (l defaultLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf("DEBUG: "+format+"\n", args...)
}

// SetLogger allows to set default package logger
func SetLogger(logger Logger) {
	log = logger
}

// NewServer creates a new Server instance
func NewServer() *Server {
	s := Server{}
	s.services = make(map[string]*service)
	s.data = make(map[string]dataRecord)
	s.Version = 1
	s.CookieName = "remote-" + randString(8)

	s.dependencies = newDataProvider()
	s.RegisterProvider(func(r *http.Request) *http.Request { return r })

	return &s
}

// Register adds an object to the API
func (s *Server) Register(rcvr interface{}) error {
	return s.register("", rcvr)
}

// RegisterProvider adds a factory method for parameters of remote methods
func (s *Server) RegisterProvider(provider interface{}) error {
	return s.dependencies.Add(provider)
}

// RegisterWithName adds an object to the API with custom name
func (s *Server) RegisterWithName(name string, rcvr interface{}) error {
	return s.register(name, rcvr)
}

// RegisterVariable adds a variable data to the API
func (s *Server) RegisterVariable(name string, rcvr interface{}) error {
	return s.registerData(name, rcvr, false)
}

// RegisterConstant adds a constant data to the API
func (s *Server) RegisterConstant(name string, rcvr interface{}) error {
	return s.registerData(name, rcvr, true)
}

func (s *Server) registerData(name string, rcvr interface{}, isConstant bool) error {
	if _, ok := s.data[name]; ok {
		return errors.New("Name already used")
	}

	if isConstant {
		s.data[name] = dataRecord{isConstant: true, value: rcvr}
	} else {
		s.data[name] = dataRecord{isConstant: false, rtype: reflect.TypeOf(rcvr)}
	}

	return nil
}

// Process starts the package processing, executing all requested methods
func (s *Server) Process(input []byte, r *http.Request) []Response {
	data := callData{}
	err := json.Unmarshal(input, &data)
	response := make([]Response, len(data))

	if err != nil {
		log.Errorf(err.Error())
		return response
	}

	res := make(chan *Response)

	for i := range data {
		data[i].dependencies = &s.dependencies
		data[i].request = r

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

	log.Debugf("Call %s.%s", call.Service(), call.Method())
	service, ok := s.services[call.Service()]
	if !ok {
		response.Error = "Unknown service"
	} else {
		service.Call(call, &response)
	}

	res <- &response
}

// JSON returns a json string representation of the end point
func (s *Server) toJSONString(key string, req *http.Request) (string, error) {
	buffer := bytes.NewBufferString("{ \"api\":{ ")

	//services
	comma := false
	for key, value := range s.services {
		jsonValue, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		if comma {
			buffer.WriteString(",")
		}
		buffer.WriteString(fmt.Sprintf("%q:%s", key, string(jsonValue)))
		comma = true
	}

	buffer.WriteString("}, \"data\":{")

	//data
	comma = false
	for key, value := range s.data {
		var err error
		var jsonValue []byte

		if value.isConstant {
			jsonValue, err = json.Marshal(value.value)
		} else {
			raw, err := s.dependencies.Value(&value.rtype, req)
			if err == nil {
				jsonValue, err = json.Marshal(raw.Interface())
			}
		}
		if err != nil {
			return "", err
		}
		if comma {
			buffer.WriteString(",")
		}
		buffer.WriteString(fmt.Sprintf("%q:%s", key, string(jsonValue)))
		comma = true
	}

	//version
	buffer.WriteString(fmt.Sprintf("}, \"key\":%q, \"version\":%d", key, s.Version))

	buffer.WriteString("}")
	return buffer.String(), nil
}

func (s *Server) register(name string, rcvr interface{}) error {
	serv := newService(rcvr)
	if name == "" {
		name = serv.name
	}
	// store the service
	s.services[name] = serv
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie(s.CookieName)
	// if cookie is not defined, create new token
	if err != nil {
		token = &http.Cookie{Name: s.CookieName, Value: randString(16)}
		http.SetCookie(w, token)
	}

	if r.Method == "GET" {
		s.serveAPI(w, r, token.Value)
		return
	}

	if r.Method != "POST" {
		s.serveError(w, errors.New("Not-supported request type"))
		return
	}

	body, err := ioutil.ReadAll(r.Body)

	// validate cookie
	key := r.Header.Get("Remote-CSRF")
	if key == "" || key != token.Value {
		log.Debugf("Invalid token %q %q", key, token.Value)
		s.serveError(w, errors.New("Invalid CSRF key"))
		return
	}

	if err != nil {
		s.serveError(w, err)
		return
	}

	res := s.Process(body, r)
	s.serveJSON(w, res)
}

func (s *Server) serveError(w http.ResponseWriter, err error) {
	text := ""
	if err != nil {
		text = err.Error()
	}
	log.Errorf(text)
	http.Error(w, text, 500)
}

func (s *Server) serveAPI(w http.ResponseWriter, req *http.Request, token string) {
	w.Header().Set("Content-type", "text/plain")
	api, _ := s.toJSONString(token, req)
	apiText(w, "remote", api)
}

func (s *Server) serveJSON(w http.ResponseWriter, res []Response) {
	w.Header().Set("Content-type", "text/json")
	out, _ := json.Marshal(res)
	w.Write(out)
}
