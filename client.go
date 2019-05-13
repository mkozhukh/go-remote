package remote

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"
)

// Client structure stores info about remote methods
type Client struct {
	url      string
	services map[string]bool
	data     map[string]interface{}
}

func (c *Client) GetData(name string) (interface{}, bool) {
	data, ok := c.data[name]
	return data, ok
}

func (c *Client) Call(name string, args []interface{}) (interface{}, error) {
	res, err := c.fetchResponse("")
	if err != nil {
		return nil, err
	}

	return res.Data, nil
}

// NewClient creates a new Client instance
func NewClient(url string) *Client {
	c := Client{}
	c.url = url

	c.fetchAPI()

	return &c
}

func (c *Client) fetchResponse(data string) (Response, error) {
	res := Response{}

	body, err := fetchBytes(c.url)
	if err != nil {
		return res, err
	}

	jsonErr := json.Unmarshal(body, &data)
	if jsonErr != nil {
		return res, jsonErr
	}

	if res.Error != "" {
		return res, errors.New(res.Error)
	}

	return res, nil
}

func (c *Client) fetchAPI() (map[string]interface{}, error) {
	body, err := fetchBytes(c.url)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	jsonErr := json.Unmarshal(body, &data)
	if jsonErr != nil {
		return nil, jsonErr
	}

	return data, nil
}

func fetchBytes(url string) ([]byte, error) {
	fetch := http.Client{
		Timeout: time.Second * 5, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-type", "application/json")

	res, err := fetch.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
