package go_remote

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"time"
)

type Client struct {
	Send   chan []byte
	Server *Server

	conn *websocket.Conn
	ctx  context.Context
}

const pongWait = 60 * time.Second
const pingPeriod = (pongWait * 9) / 10
const maxMessageSize = 4000
const writeWait = 10 * time.Second

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func (c *Client) readPump() {
	defer func() {
		c.Server.Events.Subscribe(Subscription{c, "", false})
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

	res := c.Server.Process(message, c.ctx)
	if len(res) < 1 {
		log.Errorf("somehow process doesn't return results")
		return
	}

	resText, err := json.Marshal(&res[0])
	if err != nil {
		log.Errorf("can't parse result to JSON", err)
		return
	}

	c.Send <- resText
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
