package agent

import (
	"fmt"
	"strconv"

	"github.com/go-resty/resty/v2"

)

type Client struct {
	baseURL    string
	client *resty.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		client: resty.New(),
	}
}

// SendGauge отправляет gauge-метрику на сервер
func (c *Client) SendGauge(name string, value float64) error {
	url := fmt.Sprintf("%s/update/gauge/%s/%s", c.baseURL, name, strconv.FormatFloat(value, 'f', -1, 64))

	resp, err := c.client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}

// SendCounter отправляет counter-метрику на сервер
func (c *Client) SendCounter(name string, value int64) error {
	url := fmt.Sprintf("%s/update/counter/%s/%s", c.baseURL, name, strconv.FormatInt(value, 10))

	resp, err := c.client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode())
	}

	return nil
}