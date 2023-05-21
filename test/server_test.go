package test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	go_remote "github.com/mkozhukh/go-remote"
)

type StubCalck struct {
	Counter int
}
type StubResult struct {
	X int
	Y int
}

var expectedError = errors.New("expected error")

func (c StubCalck) Add(x int, y int) int { return x + y }
func (c StubCalck) AddComplex(x StubResult, y int) (StubResult, error) {
	result := StubResult{}

	if y == 0 {
		return result, expectedError
	}

	result.X = x.X + y
	result.Y = x.Y + y

	return result, nil
}

func compareJSON(actualString string, expectedString string) bool {
	var actualValue interface{}
	var expectedValue interface{}

	err := json.Unmarshal([]byte(actualString), &actualValue)
	if err != nil {
		return false
	}

	err = json.Unmarshal([]byte(expectedString), &expectedValue)
	if err != nil {
		return false
	}

	return reflect.DeepEqual(actualValue, expectedValue)
}

func TestEmptyAPI(t *testing.T) {
	c := go_remote.NewServer()

	ctx := c.Context.FromRequest(&http.Request{})
	api := c.GetAPI(ctx)
	if !reflect.DeepEqual(api, go_remote.API{
		Services: map[string]go_remote.ServiceAPI{},
		Data:     map[string]interface{}{},
		Key:      "",
	}) {
		t.Errorf("Incorrect empty api serialization, %+v", api)
	}
}

func TestEmptyAPIWithToken(t *testing.T) {
	c := go_remote.NewServer()
	c.Context.AddProvider(func(ctx context.Context, r *http.Request) context.Context {
		return context.WithValue(ctx, go_remote.TokenValue, "123")
	})

	ctx := c.Context.FromRequest(&http.Request{})
	api := c.GetAPI(ctx)

	if !reflect.DeepEqual(api, go_remote.API{
		Services: map[string]go_remote.ServiceAPI{},
		Data:     map[string]interface{}{},
		Key:      "123",
	}) {
		t.Errorf("Incorrect empty api serialization, %+v", api)
	}
}

func TestRegisterConstant(t *testing.T) {
	c := go_remote.NewServer()

	someData := struct {
		Name   string
		Height int
	}{"Alex", 100}

	err := c.AddConstant("test1", 123)
	if err != nil {
		t.Error("AddConstant error " + err.Error())
	}
	err = c.AddConstant("test2", someData)
	if err != nil {
		t.Error("AddConstant error " + err.Error())
	}
	err = c.AddConstant("test3", &someData)
	if err != nil {
		t.Error("AddConstant error " + err.Error())
	}

	ctx := c.Context.FromRequest(&http.Request{})
	api := c.GetAPI(ctx)

	if !reflect.DeepEqual(api, go_remote.API{
		Services: map[string]go_remote.ServiceAPI{},
		Data: map[string]interface{}{
			"test1": 123,
			"test2": someData,
			"test3": someData,
		},
		Key: "",
	}) {
		t.Errorf("Incorrect api serialization, %+v", api)
	}
}

func TestRegisterVariable(t *testing.T) {
	c := go_remote.NewServer()

	type UserInfo1 struct {
		Name   string
		Height int
	}
	type UserInfo2 struct {
		Name   string
		Height int
	}

	someData := UserInfo1{"Alex", 100}
	otherData := UserInfo2{Name: "Diana", Height: 200}

	err := c.AddVariable("user1", &UserInfo1{})
	if err != nil {
		t.Error("AddConstant error " + err.Error())
	}
	err = c.AddVariable("user2", UserInfo1{})
	if err != nil {
		t.Error("AddConstant error " + err.Error())
	}
	err = c.AddVariable("user3", &UserInfo2{})
	if err != nil {
		t.Error("AddConstant error " + err.Error())
	}
	err = c.AddVariable("user4", UserInfo2{})
	if err != nil {
		t.Error("AddConstant error " + err.Error())
	}

	err = c.Dependencies.AddProvider(func(ctx context.Context) UserInfo1 {
		return someData
	})
	err = c.Dependencies.AddProvider(func(ctx context.Context) *UserInfo2 {
		return &otherData
	})

	ctx := c.Context.FromRequest(&http.Request{})
	api := c.GetAPI(ctx)

	if !reflect.DeepEqual(api, go_remote.API{
		Services: map[string]go_remote.ServiceAPI{},
		Data: map[string]interface{}{
			"user1": someData,
			"user2": someData,
			"user3": otherData,
			"user4": otherData,
		},
		Key: "",
	}) {
		t.Errorf("Incorrect api serialization, %+v", api)
	}
}

func TestRegisterService(t *testing.T) {
	c := go_remote.NewServer()

	err := c.AddService("", StubCalck{})
	if err != nil {
		t.Error("AddConstant error " + err.Error())
	}
	err = c.AddService("c2", StubCalck{})
	if err != nil {
		t.Error("AddConstant error " + err.Error())
	}

	ctx := c.Context.FromRequest(&http.Request{})
	api := c.GetAPI(ctx)

	if !reflect.DeepEqual(api, go_remote.API{
		Services: map[string]go_remote.ServiceAPI{
			"StubCalck": go_remote.ServiceAPI{
				"Add":        1,
				"AddComplex": 1,
			},
			"c2": go_remote.ServiceAPI{
				"Add":        1,
				"AddComplex": 1,
			},
		},
		Data: map[string]interface{}{},
		Key:  "",
	}) {
		t.Errorf("Incorrect api serialization, %+v", api)
	}

}

func TestProcessSingle(t *testing.T) {
	c := go_remote.NewServer()
	m := []byte("[{ \"name\":\"StubCalck.Add\", \"args\":[2,3]}]")

	err := c.AddService("", StubCalck{})
	if err != nil {
		t.Error("AddConstant error " + err.Error())
	}
	ctx := c.Context.FromRequest(&http.Request{})
	res := c.Process(m, ctx)

	if len(res) != 1 || res[0].Data.(int) != 5 {
		t.Errorf("Incorrect call result, %+v", res)
	}
}

//func TestProcessMultiple(t *testing.T) {
//	c := []byte("[{ \"name\":\"StubCalck.Add\", \"args\":[2,3]}, { \"name\":\"StubCalck.Add\", \"args\":[-2,3]}]")
//	s := remote.NewServer()
//
//	s.Register("", StubCalck{})
//	res := s.Process(c, nil, nil)
//
//	if len(res) != 2 ||
//		((res[0].Data.(int) != 5 || res[1].Data.(int) != 1) &&
//			(res[1].Data.(int) != 5 || res[0].Data.(int) != 1)) {
//		t.Errorf("Incorrect call result, %+v", res)
//	}
//}
//
//func TestProcessComplex(t *testing.T) {
//	c := []byte("[{ \"name\":\"StubCalck.AddComplex\", \"args\":[{\"X\":100,\"Y\":200},3]}]")
//	s := remote.NewServer()
//
//	s.Register("", StubCalck{})
//	res := s.Process(c, nil, nil)
//
//	if len(res) != 1 || res[0].Data.(StubResult).X != 103 || res[0].Data.(StubResult).Y != 203 {
//		t.Errorf("Incorrect call result, %+v", res)
//	}
//
//	str, err := json.Marshal(res)
//	if err != nil {
//		t.Error(err)
//		return
//	}
//
//	expected := `[{"id":"","data":{"X":103,"Y":203},"error":""}]`
//	if string(str) != expected {
//		t.Errorf("Invalid JSON result \n%q\n%q", string(str), expected)
//	}
//}
//
//func TestProcessError(t *testing.T) {
//	c := []byte("[{ \"name\":\"StubCalck.AddComplex\", \"args\":[{\"X\":100,\"Y\":200},0]}]")
//	s := remote.NewServer()
//
//	s.Register("", StubCalck{})
//	res := s.Process(c, nil, nil)
//
//	if len(res) != 1 || res[0].Error != "Expected error" {
//		t.Errorf("Wrong error response %+v", res[0])
//	}
//}
