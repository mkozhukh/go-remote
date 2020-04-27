package go_remote

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/websocket"
)

type key int

var TokenValue = key(1)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := s.Context.FromRequest(r)
	token := ctx.Value(TokenValue)
	strToken, ok := token.(string)

	requestToken := r.Header.Get("Remote-CSRF")
	isAPIListing := r.Method == "GET" && r.URL.Query().Get("ws") == ""

	// token is not defined or incorrect
	if !isAPIListing && (requestToken == "" || !ok || strToken != requestToken) {
		log.Debugf("Invalid token %q %q", strToken, requestToken)
		serveError(w, errors.New("invalid CSRF token"))
		return
	}

	if r.Method == "GET" {
		if isAPIListing {
			if strToken == "" {
				strToken = "test"
				ctx = context.WithValue(ctx, TokenValue, strToken)
				s.Context.ToResponse(w, ctx, TokenValue)
			}
			serveJSON(w, s.GetAPI(ctx))
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			serveError(w, err)
			return
			//log.Errorf("socket upgrade error: %f", err)
		}

		client := Client{Server: s, conn: conn, Send: make(chan []byte, 256), ctx: ctx}
		go client.writePump()
		go client.readPump()
		return

	}

	if r.Method != "POST" {
		serveError(w, errors.New("only post and get request types are supported"))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		serveError(w, err)
		return
	}
	res := s.Process(body, ctx)
	serveJSON(w, res)
}

func serveError(w http.ResponseWriter, err error) {
	text := err.Error()
	log.Errorf(text)
	http.Error(w, text, 500)
}

func serveJSON(w http.ResponseWriter, res interface{}) {
	w.Header().Set("Content-type", "text/json")
	out, _ := json.Marshal(res)
	w.Write(out)
}
