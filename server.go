package remote

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

// Guard is a guard function
type Guard = func(r *http.Request) bool

type apiState struct {
	methods *serviceProvider
	vars    *dataProvider
	di      *provider

	version int
	cookie  string
}

// Server structure stores all methods and data of API
type Server struct {
	*apiState
	guard interface{}
}

// NewServer creates a new Server instance
func NewServer() *Server {
	s := Server{}
	s.di = newProvider()
	s.methods = newServiceProvider(s.di)
	s.vars = newDataProvider(s.di)

	s.version = 1
	s.cookie = "remote-" + randString(8)

	return &s
}

// WithGuards creates a Server facade with guard applied
// each call to added API will be executed only if guard resolves correctly
func (s *Server) WithGuard(guard interface{}) *Server {
	n := Server{}
	n.apiState = s.apiState
	n.guard = guard
	return &n
}

// AddMethod adds an object to the API
func (s *Server) AddMethod(name string, rcvr interface{}) {
	s.methods.Add(name, rcvr, s.guard)
}

// RegisterProvider adds a factory method for parameters of remote methods
func (s *Server) AddProvider(provider interface{}) error {
	return s.di.Add(provider, s.guard)
}

// RegisterVariable adds a variable data to the API
func (s *Server) AddVariable(name string, rcvr interface{}) {
	s.vars.Add(name, rcvr, false)
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
	log.Debugf("Call %s", call.Name)
	data, err := s.methods.Value(call)

	response := Response{ID: call.ID, Data: data}
	if err != nil {
		response.Error = err.Error()
	}

	res <- &response
}

// toJSONString returns a json string representation of the api
func (s *Server) toJSONString(key string, req *http.Request, w http.ResponseWriter) ([]byte, error) {
	api := make(map[string]interface{})

	methods, err := s.methods.ToJSON()
	if err != nil {
		return nil, err
	}

	vars, err := s.vars.ToJSON(&innerCallInfo{req})
	if err != nil {
		return nil, err
	}

	api["api"] = methods
	api["data"] = vars
	api["key"] = key
	api["version"] = s.version

	return json.Marshal(api)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token, err := r.Cookie(s.cookie)
	// if cookie is not defined, create new token
	if err != nil {
		token = &http.Cookie{Name: s.cookie, Value: randString(16)}
		http.SetCookie(w, token)
	}

	// serve API
	if r.Method == "GET" {
		ctype := r.Header.Get("Accept")
		if ctype == "application/json" {
			s.jsonAPI(w, r, token.Value)
		} else {
			s.jsAPI(w, r, token.Value)
		}
		return
	}

	// process API calls
	if r.Method != "POST" {
		s.serveError(w, errors.New("not-supported request type"))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.serveError(w, err)
		return
	}

	// validate cookie
	key := r.Header.Get("Remote-CSRF")
	if key == "" || key != token.Value {
		log.Debugf("Invalid token %q %q", key, token.Value)
		s.serveError(w, errors.New("invalid token"))
		return
	}

	s.serveJSON(w, s.Process(body, r))
}

func (s *Server) serveError(w http.ResponseWriter, err error) {
	text := err.Error()

	log.Errorf(text)
	http.Error(w, text, 500)
}

func (s *Server) jsAPI(w http.ResponseWriter, req *http.Request, token string) {
	w.Header().Set("Content-type", "text/plain")
	api, err := s.toJSONString(token, req, w)
	if err != nil {
		s.serveError(w, err)
		return
	}

	apiText(w, "remote", string(api))
}

func (s *Server) jsonAPI(w http.ResponseWriter, req *http.Request, token string) {
	w.Header().Set("Content-type", "application/json")
	api, err := s.toJSONString(token, req, w)
	if err != nil {
		s.serveError(w, err)
		return
	}

	w.Write(api)
}

func (s *Server) serveJSON(w http.ResponseWriter, res []Response) {
	w.Header().Set("Content-type", "text/json")
	out, _ := json.Marshal(res)
	w.Write(out)
}
