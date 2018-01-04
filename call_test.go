package remote

import (
	"encoding/json"
	"testing"
)

type someStruct struct {
	Height int
	Name   string `json:"fullname"`
}

func TestReadArgument(t *testing.T) {
	instr := []byte("\"a\"")
	innum := []byte("12")
	instruct := []byte("{ \"height\": 100, \"fullname\":\"Alex\"}")
	c := callInfo{Name: "some.name", Args: []json.RawMessage{instr, innum, instruct}}

	num := 0
	str := ""
	strct := someStruct{}

	c.ReadArgument(0, &str)
	if str != "a" {
		t.Error("Can't parse string argument")
	}

	c.ReadArgument(1, &num)
	if num != 12 {
		t.Error("Can't parse number argument")
	}

	c.ReadArgument(2, &strct)
	if strct.Height != 100 || strct.Name != "Alex" {
		t.Error("Can't parse struct argument")
	}
}
