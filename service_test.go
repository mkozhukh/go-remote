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
	s := newService(StubCalck2{})
	thecall := callInfo{Name: "StubCalck2.Add", Args: []json.RawMessage{[]byte("2"), []byte("3")}}

	x, err := s.Call(&thecall)

	if err != nil {
		t.Error(err)
		return
	}

	if x.(int) != 5 {
		t.Errorf("Invalid call result, 5 <> %d", x)
	}
}

func TestServiceComplexCall(t *testing.T) {
	s := newService(StubCalck2{})
	thecall := callInfo{Name: "StubCalck2.AddLine", Args: []json.RawMessage{[]byte("{\"X1\":100, \"X2\":200 }"), []byte("3")}}

	x, err := s.Call(&thecall)

	if err != nil {
		t.Error(err)
		return
	}

	x1 := x.(LineStub).X1
	x2 := x.(LineStub).X2
	if x1 != 103 || x2 != 203 {
		t.Errorf("Invalid call result, 103 <> %d, 203 <> %d", x1, x2)
	}
}

func TestServiceMixedResultCall(t *testing.T) {
	s := newService(StubCalck2{})
	thecall := callInfo{Name: "StubCalck2.MixedResult", Args: []json.RawMessage{[]byte("2"), []byte("3")}}
	x, err := s.Call(&thecall)

	if err != nil {
		t.Error(err)
		return
	}

	if x != 5 {
		t.Errorf("Invalid call result, 5 <> %d", x)
	}

	thecall = callInfo{Name: "StubCalck2.MixedResult", Args: []json.RawMessage{[]byte("0"), []byte("3")}}
	x, err = s.Call(&thecall)

	if err != nil {
		if err.Error() != "Expected error" {
			t.Error(err)
		}
	} else {
		t.Error("Error was expected but was not received")
	}
}

func TestServiceSingleResultCall(t *testing.T) {
	s := newService(StubCalck2{})
	thecall := callInfo{Name: "StubCalck2.ErrorResult", Args: []json.RawMessage{[]byte("2")}}
	_, err := s.Call(&thecall)

	if err != nil {
		t.Error(err)
		return
	}

	thecall = callInfo{Name: "StubCalck2.ErrorResult", Args: []json.RawMessage{[]byte("0")}}
	_, err = s.Call(&thecall)

	if err != nil {
		if err.Error() != "Expected error" {
			t.Error(err)
		}
	} else {
		t.Error("Error was expected but was not received")
	}
}
