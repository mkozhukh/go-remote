package go_remote

import (
	"fmt"
)

type Message struct {
	Channel string      `json:"name"`
	Content interface{} `json:"value"`
}

type subscription struct {
	Client  *Client
	Channel string
	Mode    bool
}

type UserChange struct {
	ID     int  `json:"id"`
	Status bool `json:"status"`
}

type UserHandler func(u *UserChange)
type ChannelGuard func(*Message, *Client) bool

type channel struct {
	clients map[*Client]bool
}

type Hub struct {
	UserHandler UserHandler

	users    map[int]int
	channels map[string]channel
	filters  map[string]ChannelGuard

	publish   chan Message
	subscribe chan subscription
	register  chan UserChange
}

func newHub() *Hub {
	return &Hub{
		UserHandler: func(u *UserChange) {},

		publish:   make(chan Message),
		subscribe: make(chan subscription),
		register:  make(chan UserChange),

		filters:  make(map[string]ChannelGuard),
		channels: make(map[string]channel),
		users:    make(map[int]int),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case sub := <-h.subscribe:
			h.onSubscribe(&sub)
		case m := <-h.publish:
			h.onPublish(&m)
		case u := <-h.register:
			h.onRegister(&u)
		}
	}
}

func (h *Hub) AddGuard(name string, filter func(*Message, *Client) bool) {
	h.filters[name] = filter
}

func (h *Hub) Subscribe(channel string, c *Client) {
	h.subscribe <- subscription{c, channel, true}
}

func (h *Hub) UnSubscribe(channel string, c *Client) {
	h.subscribe <- subscription{c, channel, false}
}

func (h *Hub) Publish(name string, data interface{}) {
	h.publish <- Message{Channel: name, Content: data}
}

func (h *Hub) UserIn(id int) {
	h.register <- UserChange{ID: id, Status: true}
}

func (h *Hub) UserOut(id int) {
	h.register <- UserChange{ID: id, Status: false}
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

func (h *Hub) onPublish(m *Message) {
	ch, ok := h.channels[m.Channel]
	filter, hasFilter := h.filters[m.Channel]

	if ok {
		for c := range ch.clients {
			if !hasFilter || filter(m, c) {
				c.SendMessage("event", m)
			}
		}
	}
}

func (h *Hub) onRegister(u *UserChange) {
	c := h.users[u.ID]
	if u.Status {
		if c == 0 {
			h.UserHandler(u)
		}

		c += 1
	} else {
		if c <= 1 {
			h.UserHandler(u)
			delete(h.users, u.ID)
			return
		} else {
			c -= 1
		}
	}

	h.users[u.ID] = c
}

func (h *Hub) LogState() string {
	out := ""
	for name, ch := range h.channels {
		out += fmt.Sprintf("%s [%d]\n", name, len(ch.clients))
	}

	return out
}
