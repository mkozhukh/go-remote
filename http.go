package go_remote

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/mkozhukh/msgpack/v5"
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

type StatusInfo struct {
	Hub HubStatus
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, err := s.Connect(r)
	if err != nil {
		serveError(w, err)
		return
	}

	isSocketStart := r.Method == "GET" && r.URL.Query().Get("ws") != ""
	if r.Method == "GET" && !isSocketStart {
		if s.config.MessagePack {
			serveMessagePack(w, s.GetAPI(ctx))
		} else {
			serveJSON(w, s.GetAPI(ctx))
		}
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
		cid, cidExists := ctx.Value(ConnectionValue).(int)
		if !cidExists {
			cid := nextId()
			ctx = context.WithValue(ctx, ConnectionValue, cid)
		}

		client := Client{Server: s, conn: conn, Send: make(chan []byte, 256), User: userID, ConnID: cid}
		client.ctx = ctx

		go client.Start()
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		serveError(w, err)
		return
	}
	res := s.Process(body, ctx)

	if s.config.MessagePack {
		serveMessagePack(w, res)
	} else {
		serveJSON(w, res)
	}
}

// func (s *Server) ServeStatus(w http.ResponseWriter, _ *http.Request) {
// 	serveJSON(w, StatusInfo{Hub: *s.Events.Status()})
// }

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

func serveMessagePack(w http.ResponseWriter, res interface{}) {
	w.Header().Set("Content-type", "application/x-msgpack")
	out, _ := msgpack.Marshal(res)
	w.Write(out)
}

var idCounter ConnectionID

func nextId() ConnectionID {
	idCounter += 1
	return idCounter
}
