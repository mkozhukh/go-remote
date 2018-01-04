package remote

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpRequestAPI(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	s := NewServer()
	s.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode != 200 {
		t.Errorf("Error result code %d", res.StatusCode)
	}

	text, _ := ioutil.ReadAll(res.Body)
	if len(text) != 2046 {
		t.Errorf("Wrong response: %q (%d)", text, len(text))
	}
}

func TestHttpPost(t *testing.T) {
	r := httptest.NewRequest("POST", "/", bytes.NewBufferString(`
		[{ "name":"x.Add", "args":[2,3]}]
	`))
	ck := http.Cookie{Name: "test", Value: "test"}
	r.AddCookie(&ck)
	r.Header.Set("Remote-CSRF", "test")

	w := httptest.NewRecorder()

	s := NewServer()
	s.CookieName = "test"
	s.RegisterName("x", StubCalck2{})
	s.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode != 200 {
		t.Errorf("Error result code %d", res.StatusCode)
	}

	text, _ := ioutil.ReadAll(res.Body)
	var data []map[string]interface{}
	err := json.Unmarshal(text, &data)

	if err != nil {
		t.Error(err)
	}

	if len(data) != 1 || data[0]["data"].(float64) != 5 {
		t.Errorf("Wrong response: %v", data)
	}
}

func TestHttpPostDI(t *testing.T) {
	r := httptest.NewRequest("POST", "/", bytes.NewBufferString(`
		[{ "name":"x.MirrorURL", "args":[2,3]}]
	`))
	ck := http.Cookie{Name: "test", Value: "test"}
	r.AddCookie(&ck)
	r.Header.Set("Remote-CSRF", "test")

	w := httptest.NewRecorder()

	s := NewServer()
	s.CookieName = "test"
	s.RegisterName("x", StubCalck2{})
	s.ServeHTTP(w, r)

	res := w.Result()
	if res.StatusCode != 200 {
		t.Errorf("Error result code %d", res.StatusCode)
		return
	}

	text, _ := ioutil.ReadAll(res.Body)
	var data []map[string]interface{}
	err := json.Unmarshal(text, &data)

	if err != nil {
		t.Error(err)
		return
	}

	if len(data) != 1 || data[0]["data"].(string) != "/" {
		t.Errorf("Wrong response: %v", data)
	}
}
