package go_remote

import (
	"fmt"
)

type Message struct {
	Channel string      `json:"name"`
	Content interface{} `json:"value"`
	Clients []ConnectionID
}

type subscription struct {
	Client  *Socket
	Channel string
	Mode    bool
}

type UserChange struct {
	ID         int  `json:"id"`
	Connection int  `json:"-"`
	Status     bool `json:"status"`
}

type HubStatus struct {
	Users    map[int]int
	Channels map[string]ChannelStatus
}

type ChannelStatus struct {
	Subscribed []int
}

type UserHandler func(u *UserChange)
type ChannelGuard func(*Message, *Socket) bool

type channel struct {
	clients map[*Socket]bool
}

type Hub struct {
	UserHandler UserHandler
	ConnHandler UserHandler

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
		ConnHandler: func(u *UserChange) {},

		publish:   make(chan Message),
		subscribe: make(chan subscription),
		register:  make(chan UserChange),

		filters:  make(map[string]ChannelGuard),
		channels: make(map[string]channel),
		users:    make(map[int]int),
	}
}

func (h *Hub) Status() *HubStatus {
	info := make(map[string]ChannelStatus)

	for name, ch := range h.channels {
		subs := make([]int, 0, len(ch.clients))
		for cl := range ch.clients {
			subs = append(subs, cl.User)
		}
		info[name] = ChannelStatus{Subscribed: subs}
	}
	return &HubStatus{
		Users:    h.users,
		Channels: info,
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

func (h *Hub) AddGuard(name string, filter func(*Message, *Socket) bool) {
	h.filters[name] = filter
}

func (h *Hub) Subscribe(channel string, c *Socket) {
	h.subscribe <- subscription{c, channel, true}
}

func (h *Hub) UnSubscribe(channel string, c *Socket) {
	h.subscribe <- subscription{c, channel, false}
}

func (h *Hub) Publish(name string, data interface{}, clients ...ConnectionID) {
	h.publish <- Message{Channel: name, Content: data, Clients: clients}
}

func (h *Hub) UserIn(id, device int) {
	h.register <- UserChange{ID: id, Connection: device, Status: true}
}

func (h *Hub) UserOut(id, conn int) {
	h.register <- UserChange{ID: id, Connection: conn, Status: false}
}

func (h *Hub) onSubscribe(sub *subscription) {
	if !sub.Mode {
		if sub.Channel == "" {
			//unsubscribe from all
			for name := range h.channels {
				h.onUnSubscribe(name, sub.Client)
			}
		} else {
			h.onUnSubscribe(sub.Channel, sub.Client)
		}

		return
	}

	ch, ok := h.channels[sub.Channel]
	if !ok {
		ch = channel{clients: make(map[*Socket]bool)}
		h.channels[sub.Channel] = ch
	}

	ch.clients[sub.Client] = true
}

func (h *Hub) onUnSubscribe(channel string, client *Socket) {
	ch, ok := h.channels[channel]
	if !ok {
		return
	}

	delete(ch.clients, client)
	if len(ch.clients) == 0 {
		delete(h.channels, channel)
	}
}

func (h *Hub) onPublish(m *Message) {
	ch, ok := h.channels[m.Channel]
	filter, hasFilter := h.filters[m.Channel]

	if ok {
		for c := range ch.clients {
			if !hasFilter || filter(m, c) {
				if len(m.Clients) != 0 {
					for _, x := range m.Clients {
						if x == ConnectionID(c.ConnID) {
							c.SendMessage("event", m)
						}
					}
				} else {
					c.SendMessage("event", m)
				}
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
	h.ConnHandler(u)
}

func (h *Hub) LogState() string {
	out := ""
	for name, ch := range h.channels {
		out += fmt.Sprintf("%s [%d]\n", name, len(ch.clients))
	}

	return out
}
