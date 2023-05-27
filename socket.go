package go_remote

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mkozhukh/msgpack/v5"
)

type ConnectionID int64

type Client struct {
	Send   chan []byte
	Server *Server
	User   int
	ConnID int

	conn *websocket.Conn
	ctx  context.Context
}

type ResponseMessage struct {
	Action string      `json:"action"`
	Body   interface{} `json:"body,omitempty"`
}

type RequestMessage struct {
	Action string `json:"action"`
	Name   string `json:"name"`
}

type RequestMessageJSON struct {
	RequestMessage
	Body json.RawMessage `json:"body,omitempty"`
}

type RequestMessageMessagePack struct {
	RequestMessage
	Body msgpack.RawMessage `json:"body,omitempty"`
}

type RequestMessageCommon interface {
	Info() *RequestMessage
	GetBody() []byte
	Unmarshal([]byte) error
}

func (r *RequestMessageJSON) Unmarshal(v []byte) error {
	return json.Unmarshal(v, r)
}

func (r *RequestMessageMessagePack) Unmarshal(v []byte) error {
	return msgpack.Unmarshal(v, r)
}

func (r RequestMessageJSON) Info() *RequestMessage {
	return &r.RequestMessage
}

func (r RequestMessageMessagePack) Info() *RequestMessage {
	return &r.RequestMessage
}

func (r *RequestMessageJSON) GetBody() []byte {
	return r.Body
}

func (r *RequestMessageMessagePack) GetBody() []byte {
	return r.Body
}

const pongWait = 60 * time.Second
const pingPeriod = (pongWait * 9) / 10

var MaxSocketMessageSize = 4000

const writeWait = 10 * time.Second

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func (c *Client) Start() {
	go c.readPump()
	go c.writePump()

	c.Server.Events.UserIn(c.User, c.ConnID)
	c.SendMessage("start", c.ConnID)
}

func (c *Client) Context() context.Context {
	return c.ctx
}

func (c *Client) SendMessage(name string, body interface{}) {
	msg := &ResponseMessage{Action: name, Body: body}

	var err error
	var m []byte
	if c.Server.config.MessagePack {
		m, err = msgpack.Marshal(msg)
	} else {
		m, err = json.Marshal(msg)
	}

	if err != nil {
		log.Error("error while marshalling message", "err", err)
		return
	}
	c.Send <- m
}

func (c *Client) readPump() {
	defer func() {
		c.Server.Events.UserOut(c.User, c.ConnID)
		c.Server.Events.UnSubscribe("", c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(int64(MaxSocketMessageSize))
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("websocket error", "err", err)
			}
			break
		}

		go c.process(message)
	}
}

func (c *Client) process(message []byte) {
	var m RequestMessageCommon
	if c.Server.config.MessagePack {
		m = &RequestMessageMessagePack{}
		// log.Debug("message", "msg", hex.Dump(message))
	} else {
		m = &RequestMessageJSON{}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		// log.Debug("message", "msg", string(message))
	}

	err := m.Unmarshal(message)
	if err != nil {
		log.Error("can't unmarshal message", "msg", message, "err", err)
		return
	}

	info := m.Info()
	if info.Action == "subscribe" {
		c.Server.Events.Subscribe(info.Name, c)
	}

	if info.Action == "unsubscribe" {
		c.Server.Events.UnSubscribe(info.Name, c)
	}

	if info.Action == "call" {
		res := c.Server.Process(m.GetBody(), c.ctx)
		if len(res) < 1 {
			log.Error("somehow process doesn't return results")
			return
		}

		c.SendMessage("result", &res)
	}
}

func (c *Client) writePump() {
	msgtype := websocket.TextMessage
	if c.Server.config.MessagePack {
		msgtype = websocket.BinaryMessage
	}

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// log.Debug("send message", "msg", hex.Dump(message))

			w, err := c.conn.NextWriter(msgtype)
			if err != nil {
				return
			}
			_, err = w.Write(message)
			if err != nil {
				return
			}
			err = w.Close()
			if err != nil {
				return
			}

			// Add queued messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w, err := c.conn.NextWriter(msgtype)
				if err != nil {
					return
				}
				_, err = w.Write(<-c.Send)
				if err != nil {
					return
				}
				err = w.Close()
				if err != nil {
					return
				}
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
