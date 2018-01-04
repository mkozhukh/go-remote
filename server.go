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
	data         map[string]interface{}
	dependencies map[reflect.Type]reflect.Value
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
	s.data = make(map[string]interface{})
	s.Version = 1
	s.CookieName = "remote-" + randString(8)

	s.dependencies = make(map[reflect.Type]reflect.Value)
	s.RegisterProvider(func(r *http.Request) *http.Request { return r })

	return &s
}

func getRequest() {

}

// Register adds an object to the API
func (s *Server) Register(rcvr interface{}) error {
	return s.register("", rcvr)
}

// RegisterProvider adds a factory method for parameters of remote methods
func (s *Server) RegisterProvider(provider interface{}) error {
	pType := reflect.TypeOf(provider)
	if pType.Kind() != reflect.Func || pType.NumOut() != 1 || pType.NumIn() != 1 || pType.In(0) != requestType {
		msg := "Invalid parameter for RegisterProvider, function factory is expected"
		log.Errorf(msg)
		return errors.New(msg)
	}

	s.dependencies[pType.Out(0)] = reflect.ValueOf(provider)
	return nil
}

// RegisterName adds an object to the API with custom name
func (s *Server) RegisterName(name string, rcvr interface{}) error {
	return s.register(name, rcvr)
}

// RegisterData adds a constant data to the API
func (s *Server) RegisterData(name string, rcvr interface{}) error {
	if _, ok := s.data[name]; ok {
		return errors.New("Name already used")
	}
	s.data[name] = rcvr
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

	for i := range data {
		data[i].dependencies = s.dependencies
		data[i].request = r

		res, err := s.Call(data[i])
		response[i].ID = data[i].ID
		if err != nil {
			response[i].Error = err.Error()
		} else {
			response[i].Data = res
		}
	}

	return response
}

// Call allows to execute some Servers's method
func (s *Server) Call(call *callInfo) (interface{}, error) {
	log.Debugf("Call %s.%s", call.Service(), call.Method())
	service, ok := s.services[call.Service()]
	if !ok {
		return nil, errors.New("Unknown service")
	}

	return service.Call(call)
}

// JSON returns a json string representation of the end point
func (s *Server) JSON(key string) (string, error) {
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
		s.serveAPI(w, token.Value)
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

func (s *Server) serveAPI(w http.ResponseWriter, token string) {
	w.Header().Set("Content-type", "text/plain")
	api, _ := s.JSON(token)
	apiText(w, "remote", api)
}

func (s *Server) serveJSON(w http.ResponseWriter, res []Response) {
	w.Header().Set("Content-type", "text/json")
	out, _ := json.Marshal(res)
	w.Write(out)
}
