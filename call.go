package go_remote

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/mkozhukh/msgpack/v5"
)

// callData stores information about the remote call
type callData interface {
	Size() int
	At(index int) callInfoCommon
	Unmarshal([]byte) error
}

type callInfoCommon interface {
	ReadArgument(index int, args interface{}) error
	Info() *callInfo
}

type callArgs interface {
	ReadArgument(index int, args interface{}) error
}

type callDataJSON []*callInfoJSON

func (c callDataJSON) Size() int {
	return len(c)
}

func (c callDataJSON) At(index int) callInfoCommon {
	return c[index]
}

func (c callDataJSON) Unmarshal(v []byte) error {
	return json.Unmarshal(v, &c)
}

type callDataMessagePack []*callInfoMessagePack

func (c callDataMessagePack) Size() int {
	return len(c)
}

func (c callDataMessagePack) At(index int) callInfoCommon {
	return c[index]
}

func (c callDataMessagePack) Unmarshal(v []byte) error {
	return msgpack.Unmarshal(v, &c)
}

type CallArgs interface {
}

type callInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	dependencies *dependencyStore
	ctx          context.Context
	service      string
	method       string
}

type callInfoJSON struct {
	callInfo
	Args []json.RawMessage `json:"args"`
}

func (c *callInfo) Init(ctx context.Context, dependencies *dependencyStore) {
	parts := strings.Split(c.Name, ".")
	c.service = parts[0]
	c.method = parts[1]

	c.ctx = ctx
	c.dependencies = dependencies
}

func (c *callInfoJSON) Info() *callInfo {
	return &c.callInfo
}

func (c *callInfoMessagePack) Info() *callInfo {
	return &c.callInfo
}

// readArgument fills the request object for the RPC method.
func (c *callInfoJSON) ReadArgument(index int, args interface{}) error {
	if index >= len(c.Args) {
		return errors.New("invalid number of parameters")
	}

	return json.Unmarshal(c.Args[index], args)
}

type callInfoMessagePack struct {
	callInfo
	Args []msgpack.RawMessage `json:"args"`
}

// readArgument fills the request object for the RPC method.
func (c *callInfoMessagePack) ReadArgument(index int, args interface{}) error {
	if index >= len(c.Args) {
		return errors.New("invalid number of parameters")
	}

	return msgpack.Unmarshal(c.Args[index], args)
}
