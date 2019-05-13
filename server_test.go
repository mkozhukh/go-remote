package remote

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

type StubCalck struct {
	Counter int
}
type StubResult struct {
	X int
	Y int
}

func (c StubCalck) Add(x int, y int) int { return x + y }
func (c StubCalck) AddComplex(x StubResult, y int) (StubResult, error) {
	result := StubResult{}

	if y == 0 {
		return result, errors.New("Expected error")
	}

	result.X = x.X + y
	result.Y = x.Y + y

	return result, nil
}
func compareJSON(actualString string, expectedString string) bool {
	var aval interface{}
	var eval interface{}

	// this is guarded by prettyPrint
	json.Unmarshal([]byte(actualString), &aval)
	json.Unmarshal([]byte(expectedString), &eval)

	return reflect.DeepEqual(aval, eval)
}

func TestNewServer(t *testing.T) {
	c := NewServer()
	text, _ := c.toJSONString("1", nil, nil)
	if string(text) != `{"api":{},"data":{},"key":"1","version":1}` {
		t.Errorf("Incorrect version serialization, %s", text)
	}
}

func TestRegisterData(t *testing.T) {
	c := NewServer()

	someData := struct {
		Name   string
		Height int
	}{"Alex", 100}

	c.AddVariable("test1", func() int { return 123 })
	c.AddVariable("test2", func() interface{} { return someData })
	c.AddVariable("test3", func() interface{} { return &someData })

	raw := c.vars.ToHashMap(&innerCallInfo{nil})
	text, err := json.Marshal(raw)
	if err != nil {
		t.Error(err)
		return
	}

	if !compareJSON(string(text), `{"test1":123,"test2":{"Name":"Alex","Height":100},"test3":{"Name":"Alex","Height":100}}`) {
		t.Errorf("Incorrect data serialization, %s", text)
	}
}

func TestRegisterName(t *testing.T) {
	c := NewServer()

	c.AddMethod("", StubCalck{})
	c.AddMethod("c2", StubCalck{})

	raw := c.methods.ToHashMap(nil)
	text, err := json.Marshal(raw)
	if err != nil {
		t.Error(err)
		return
	}

	expected := `{"StubCalck":{"Add":1,"AddComplex":1},"c2":{"Add":1,"AddComplex":1}}`
	if !compareJSON(string(text), expected) {
		t.Errorf("Incorrect api serialization\n%s\n%s", text, expected)
	}
}

func TestProcessSingle(t *testing.T) {
	c := []byte("[{ \"name\":\"StubCalck.Add\", \"args\":[2,3]}]")
	s := NewServer()

	s.AddMethod("", StubCalck{})
	res := s.Process(c, nil)

	if len(res) != 1 || res[0].Data.(int) != 5 {
		t.Errorf("Incorrect api serialization, %+v", res)
	}
}

func TestProcessMultiple(t *testing.T) {
	c := []byte("[{ \"name\":\"StubCalck.Add\", \"args\":[2,3]}, { \"name\":\"StubCalck.Add\", \"args\":[-2,3]}]")
	s := NewServer()

	s.AddMethod("", StubCalck{})
	res := s.Process(c, nil)

	if len(res) != 2 ||
		((res[0].Data.(int) != 5 || res[1].Data.(int) != 1) &&
			(res[1].Data.(int) != 5 || res[0].Data.(int) != 1)) {
		t.Errorf("Incorrect call result, %+v", res)
	}
}

func TestProcessComplex(t *testing.T) {
	c := []byte("[{ \"name\":\"StubCalck.AddComplex\", \"args\":[{\"X\":100,\"Y\":200},3]}]")
	s := NewServer()

	s.AddMethod("", StubCalck{})
	res := s.Process(c, nil)

	if len(res) != 1 || res[0].Data.(StubResult).X != 103 || res[0].Data.(StubResult).Y != 203 {
		t.Errorf("Incorrect call result, %+v", res)
	}

	str, err := json.Marshal(res)
	if err != nil {
		t.Error(err)
		return
	}

	expected := `[{"id":"","data":{"X":103,"Y":203},"error":""}]`
	if string(str) != expected {
		t.Errorf("Invalid JSON result \n%q\n%q", string(str), expected)
	}
}

func TestProcessError(t *testing.T) {
	c := []byte("[{ \"name\":\"StubCalck.AddComplex\", \"args\":[{\"X\":100,\"Y\":200},0]}]")
	s := NewServer()

	s.AddMethod("", StubCalck{})
	res := s.Process(c, nil)

	if len(res) != 1 || res[0].Error != "Expected error" {
		t.Errorf("Wrong error response %+v", res[0])
	}
}
