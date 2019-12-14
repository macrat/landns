package client

import (
	"net/url"
	"strings"
	"io"
	"io/ioutil"
	"fmt"
	"net/http"

	"github.com/macrat/landns/lib-landns"
)

type Client struct {
	endpoint *url.URL
	client   *http.Client
}

func New(endpoint *url.URL) Client {
	return Client{
		endpoint: endpoint,
		client: &http.Client{},
	}
}

func (c Client) do(method, path string, body fmt.Stringer) (response landns.DynamicRecordSet, err error) {
	u, err := c.endpoint.Parse(path)
	if err != nil {
		return
	}

	us := u.String()
	if strings.HasSuffix(us, "/") {
		us = us[:len(us)-1]
	}

	var r io.Reader
	if body != nil {
		r = strings.NewReader(body.String())
	}
	req, err := http.NewRequest(method, us, r)
	if err != nil {
		return
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	rbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return
	}

	return response, response.UnmarshalText(rbody)
}

func (c Client) Set(records landns.DynamicRecordSet) error {
	_, err := c.do("POST", "", records)
	return err
}

func (c Client) Remove(id int) error {
	_, err := c.do("DELETE", fmt.Sprintf("id/%d", id), nil)
	return err
}

func (c Client) Get() (landns.DynamicRecordSet, error) {
	return c.do("GET", "", nil)
}

func (c Client) Glob(query string) (landns.DynamicRecordSet, error) {
	return c.do("GET", fmt.Sprintf("glob/%s", query), nil)
}
