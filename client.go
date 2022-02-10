package go_remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

type Client struct {
	config ClientConfig
}

type ClientRequest struct {
	ID   string        `json:"id"`
	Name string        `json:"name"`
	Args []interface{} `json:"args"`
}

type ClientRequestPack struct {
	data []ClientRequest
}

type ClientConfig struct {
	WebSocket bool
	Url       string
}

func NewClient(cfg ClientConfig) *Client {
	s := Client{}
	s.config = cfg

	if s.config.Url == "" {
		panic("can't create client without server url")
	}

	return &s
}

func (c *Client) List() (*API, error) {
	res, err := http.DefaultClient.Get(c.config.Url)
	if err != nil {
		return nil, fmt.Errorf("can't list api data at '%s'\n%w", c.config.Url, err)
	}

	temp, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	api := API{}
	err = json.Unmarshal(temp, &api)
	if err != nil {
		return nil, fmt.Errorf("can't parse json response:\n%s\n%w", string(temp), err)
	}

	return &api, nil
}

func (c *Client) Trigger(name string, args []interface{}) (*Response, error) {
	pack := NewClientRequestPack()
	pack.Add(name, args)
	res, err := c.TriggerPack(pack)
	if err != nil {
		return nil, err
	}

	if len(res) != 1 {
		return nil, fmt.Errorf("response has invalid format:\n%+v\n%w", res, err)
	}

	return &res[0], nil
}

func (c *Client) TriggerPack(p *ClientRequestPack) ([]Response, error) {
	temp, err := p.Marshal()
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Post(c.config.Url, "application/json", bytes.NewBuffer(temp))
	if err != nil {
		return nil, fmt.Errorf("can't connect remote server:\n%s\n%w", c.config.Url, err)
	}

	tdata, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	result := make([]Response, 0)
	err = json.Unmarshal(tdata, &result)
	if err != nil {
		return nil, fmt.Errorf("can't parse json response:\n%s\n%w", string(temp), err)
	}

	return result, nil
}

func (c *Client) Connect() (*ClientSession, error) {
	return nil, nil
}

func NewClientRequestPack() *ClientRequestPack {
	t := ClientRequestPack{}
	t.data = make([]ClientRequest, 0, 1)
	return &t
}
func (c *ClientRequestPack) Add(name string, args []interface{}) {
	id := strconv.Itoa(int(nextId()))
	c.data = append(c.data, ClientRequest{id, name, args})
}

func (c *ClientRequestPack) Marshal() ([]byte, error) {
	temp, err := json.Marshal(&c.data)
	if err != nil {
		return nil, fmt.Errorf("can't convert arguments to json:\n%+v\n%w", c.data, err)
	}

	return temp, nil
}

type ClientSession struct {
	Message chan string
	ws      int
}

func (c *ClientSession) Trigger()                {}
func (c *ClientSession) Subscribe(name string)   {}
func (c *ClientSession) Unsubscribe(name string) {}
