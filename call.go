package go_remote

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// callData stores information about the remote call
type callData []*callInfo

type callInfo struct {
	ID   string            `json:"id"`
	Name string            `json:"name"`
	Args []json.RawMessage `json:"args"`

	dependencies *dependencyStore
	ctx          context.Context
	service      string
	method       string
}

func (c *callInfo) parse() {
	parts := strings.Split(c.Name, ".")
	c.service = parts[0]
	c.method = parts[1]
}

// readArgument fills the request object for the RPC method.
func (c *callInfo) readArgument(index int, args interface{}) error {
	if index >= len(c.Args) {
		return errors.New("Invalid number of parameters")
	}

	return json.Unmarshal(c.Args[index], args)
}
