package remote

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

type StubCalck2 struct {
	Counter int
}

type LineStub struct {
	X1 int
	X2 int
}

func (c StubCalck2) Add(x int, y int) int { return x + y }
func (c StubCalck2) ErrorResult(x int) error {
	if x == 0 {
		return errors.New("Expected error")
	}
	return nil
}
func (c StubCalck2) MixedResult(x int, y int) (int, error) {
	if x > 0 {
		return x + y, nil
	}
	return 0, errors.New("Expected error")
}

func (c StubCalck2) AddLine(x LineStub, y int) (res LineStub) {
	res.X1 = x.X1 + y
	res.X2 = x.X2 + y
	return
}

func (c StubCalck2) MirrorURL(x int, y int, r *http.Request) string {
	return r.RequestURI
}

func TestServiceCall(t *testing.T) {
	s := newService(StubCalck2{}, nil)
	thecall := callInfo{Name: "StubCalck2.Add", Args: []json.RawMessage{[]byte("2"), []byte("3")}}

	c := Response{}
	s.Call(&thecall, &c)

	if c.Error != "" {
		t.Error(c.Error)
		return
	}

	if c.Data.(int) != 5 {
		t.Errorf("Invalid call result, 5 <> %d", c.Data)
	}
}

func TestServiceComplexCall(t *testing.T) {
	s := newService(StubCalck2{}, nil)
	thecall := callInfo{Name: "StubCalck2.AddLine", Args: []json.RawMessage{[]byte("{\"X1\":100, \"X2\":200 }"), []byte("3")}}

	c := Response{}
	s.Call(&thecall, &c)

	if c.Error != "" {
		t.Error(c.Error)
		return
	}

	x1 := c.Data.(LineStub).X1
	x2 := c.Data.(LineStub).X2
	if x1 != 103 || x2 != 203 {
		t.Errorf("Invalid call result, 103 <> %d, 203 <> %d", x1, x2)
	}
}

func TestServiceMixedResultCall(t *testing.T) {
	s := newService(StubCalck2{}, nil)
	thecall := callInfo{Name: "StubCalck2.MixedResult", Args: []json.RawMessage{[]byte("2"), []byte("3")}}

	c := Response{}
	s.Call(&thecall, &c)

	if c.Error != "" {
		t.Error(c.Error)
		return
	}

	if c.Data != 5 {
		t.Errorf("Invalid call result, 5 <> %d", c.Data)
	}

	thecall = callInfo{Name: "StubCalck2.MixedResult", Args: []json.RawMessage{[]byte("0"), []byte("3")}}

	c = Response{}
	s.Call(&thecall, &c)

	if c.Error != "" {
		if c.Error != "Expected error" {
			t.Error(c.Error)
		}
	} else {
		t.Error("Error was expected but was not received")
	}
}

func TestServiceSingleResultCall(t *testing.T) {
	s := newService(StubCalck2{}, nil)
	thecall := callInfo{Name: "StubCalck2.ErrorResult", Args: []json.RawMessage{[]byte("2")}}
	c := Response{}
	s.Call(&thecall, &c)

	if c.Error != "" {
		t.Error(c.Error)
		return
	}

	thecall = callInfo{Name: "StubCalck2.ErrorResult", Args: []json.RawMessage{[]byte("0")}}
	c = Response{}
	s.Call(&thecall, &c)

	if c.Error != "" {
		if c.Error != "Expected error" {
			t.Error(c.Error)
		}
	} else {
		t.Error("Error was expected but was not received")
	}
}
