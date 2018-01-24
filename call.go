package remote

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// callData stores information about the remote call
type callData []*callInfo

// Response handles results of remote calls
type Response struct {
	ID    string      `json:"id"`
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
}

type callInfo struct {
	ID   string            `json:"id"`
	Name string            `json:"name"`
	Args []json.RawMessage `json:"args"`

	dependencies *dataProvider
	request      *http.Request
	writer       http.ResponseWriter
	service      string
	method       string
}

// ReadArgument fills the request object for the RPC method.
func (c *callInfo) ReadArgument(index int, args interface{}) error {
	if index >= len(c.Args) {
		return errors.New("Invalid number of parameters")
	}

	return json.Unmarshal(c.Args[index], args)
}

func (c *callInfo) parseName() {
	parts := strings.Split(c.Name, ".")
	c.service = parts[0]
	c.method = parts[1]
}

func (c *callInfo) Service() string {
	c.parseName()
	return c.service
}

func (c *callInfo) Method() string {
	c.parseName()
	return c.method
}
