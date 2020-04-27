package go_remote

import (
	"fmt"
)

type message struct {
	Channel string
	Content interface{}
}

type subscription struct {
	Client  *Client
	Channel string
	Mode    bool
}

type channel struct {
	clients map[*Client]bool
}

type Hub struct {
	clients  map[*Client]bool
	channels map[string]channel

	publish   chan message
	subscribe chan subscription
}

func newHub() *Hub {
	return &Hub{
		publish:   make(chan message),
		subscribe: make(chan subscription),
		channels:  make(map[string]channel),
		clients:   make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case sub := <-h.subscribe:
			h.onSubscribe(&sub)
		case m := <-h.publish:
			h.onPublish(&m)
		}
	}
}

func (h *Hub) Subscribe(channel string, c *Client) {
	h.subscribe <- subscription{c, channel, true}
}

func (h *Hub) UnSubscribe(channel string, c *Client) {
	h.subscribe <- subscription{c, channel, false}
}

func (h *Hub) Publish(name string, data interface{}) {
	h.publish <- message{Channel:name, Content: data}
}

func (h *Hub) onSubscribe(sub *subscription) {
	ch, ok := h.channels[sub.Channel]
	if !ok {
		if !sub.Mode {
			// unsubscribe from non-existing channel
			return
		}
		ch = channel{clients: make(map[*Client]bool)}
		h.channels[sub.Channel] = ch
	}

	if sub.Mode {
		ch.clients[sub.Client] = true
	} else {
		delete(ch.clients, sub.Client)
		if len(ch.clients) == 0 {
			delete(h.channels, sub.Channel)
		}
	}
}

func (h *Hub) onPublish(m *message) {
	ch, ok := h.channels[m.Channel]
	if ok {
		for c := range ch.clients {
			c.SendMessage("event", m.Content)
		}
	}
}

func (h *Hub) LogState() string {
	out := ""
	for name, ch := range h.channels {
		out += fmt.Sprintf("%s [%d]\n", name, len(ch.clients))
	}

	return out
}
