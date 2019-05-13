package remote

import (
	"encoding/json"
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
	innerCallInfo

	ID   string            `json:"id"`
	Name string            `json:"name"`
	Args []json.RawMessage `json:"args"`

	index int
}

// ReadArgument fills the request object for the RPC method.
func (c *callInfo) NextArgument(args interface{}) error {
	if c.index >= len(c.Args) {
		return nil
	}

	err := json.Unmarshal(c.Args[c.index], args)
	c.index++

	return err
}

func (c *callInfo) SplitName() string {
	parts := strings.Split(c.Name, ".")

	if len(parts) == 1 {
		return "";
	}

	c.Name = parts[1]
	return parts[0]
}



type innerCallInfo struct {
	request *http.Request
}
func (c *innerCallInfo) Request() *http.Request{
	return c.request
}
// ReadArgument fills the request object for the RPC method.
func (c *innerCallInfo) NextArgument(args interface{}) error {
	return nil
}

type callState interface {
	Request() *http.Request
	NextArgument(args interface{}) error
}