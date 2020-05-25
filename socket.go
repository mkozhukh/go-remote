package go_remote

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Send   chan []byte
	Server *Server

	conn *websocket.Conn
	ctx  context.Context
}

type ResponseMessage struct {
	Action string      `json:"action"`
	Body   interface{} `json:"body,omitempty"`
}

type RequestMessage struct {
	Action  string          `json:"action"`
	Channel string          `json:"channel"`
	Body    json.RawMessage `json:"body,omitempty"`
}

const pongWait = 60 * time.Second
const pingPeriod = (pongWait * 9) / 10
const maxMessageSize = 4000
const writeWait = 10 * time.Second

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func (c *Client) Start() {
	go c.readPump()
	go c.writePump()

	c.SendMessage("start", nil)
}

func (c *Client) SendMessage(name string, body interface{}) {
	m, _ := json.Marshal(&ResponseMessage{Action: name, Body: body})
	c.Send <- m
}

func (c *Client) readPump() {
	defer func() {
		c.Server.Events.UnSubscribe("", c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("websocket error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		go c.process(message)
	}
}

func (c *Client) process(message []byte) {
	m := RequestMessage{}
	err := json.Unmarshal(message, &m)
	if err != nil {
		log.Errorf("invalid message: %s", message)
		log.Errorf(err.Error())
		return
	}

	if m.Action == "subscribe" {
		c.Server.Events.Subscribe(m.Channel, c)
	}

	if m.Action == "unsubscribe" {
		c.Server.Events.UnSubscribe(m.Channel, c)
	}

	if m.Action == "remote" {
		res := c.Server.Process(m.Body, c.ctx)
		if len(res) < 1 {
			log.Errorf("somehow process doesn't return results")
			return
		}

		c.SendMessage("result", &res)
	}
}

func (c *Client) writePump() {
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

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}