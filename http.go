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

var UserValue = key(1)
var ConnectionValue = key(2)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.Connect(r.Context())
	if err != nil {
		serveError(w, err)
		return
	}

	isSocketStart := r.Method == "GET" && r.URL.Query().Get("ws") != ""
	if r.Method == "GET" && !isSocketStart {
		serveJSON(w, s.GetAPI(ctx))
		return
	}

	if !isSocketStart && r.Method != "POST" {
		serveError(w, errors.New("only post and get request types are supported"))
		return
	}

	if isSocketStart {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			serveError(w, err)
			return
		}

		userID, _ := ctx.Value(UserValue).(int)
		id := nextId()
		client := Client{Server: s, conn: conn, Send: make(chan []byte, 256), User: userID, ConnID:id }
		client.ctx = context.WithValue(ctx, ConnectionValue, id)
		go client.Start()
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

var idCounter int64
func nextId() int64 {
	idCounter += 1
	return idCounter
}