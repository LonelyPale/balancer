package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/lonelypale/balancer"
)

type HttpClient struct {
	client   *http.Client
	balancer balancer.Balancer
}

func New(opts balancer.Options) (*HttpClient, error) {
	builder := balancer.Get(opts.Type)
	if builder == nil {
		return nil, fmt.Errorf("unknown load balance type: %s", opts.Type)
	}

	if opts.Timeout <= 0 {
		opts.Timeout = 30
	}
	client := &http.Client{Timeout: time.Duration(opts.Timeout) * time.Second}

	loadBalancing := builder.Build(client, &opts)
	if loadBalancing == nil {
		return nil, fmt.Errorf("%s load balance failed to build", opts.Type)
	}

	return &HttpClient{
		client:   client,
		balancer: loadBalancing,
	}, nil
}

func (h *HttpClient) Get(url string, result interface{}) error {
	return h.request("GET", url, nil, nil, result)
}

func (h *HttpClient) GetWithHeader(url string, header map[string]string, result interface{}) error {
	return h.request("GET", url, header, nil, result)
}

func (h *HttpClient) Post(url string, payload []byte, result interface{}) error {
	return h.request("POST", url, nil, payload, result)
}

func (h *HttpClient) PostWithHeader(url string, header map[string]string, payload []byte, result interface{}) error {
	return h.request("POST", url, header, payload, result)
}

func (h *HttpClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := h.balancer.Do(req)

	//Failure retry
	if err != nil || (resp != nil && resp.StatusCode >= 400) {
		for i := 0; i < 3; i++ {
			resp, err = h.balancer.Do(req)
			if err == nil && (resp != nil && resp.StatusCode < 400) { //success
				return resp, err
			}
		}
	}

	return resp, err
}

func (h *HttpClient) Request(method, url string, header map[string]string, payload []byte, result interface{}) error {
	return h.request(method, url, header, payload, result)
}

func (h *HttpClient) request(method, url string, header map[string]string, payload []byte, result interface{}) error {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	// set Content-Type in advance, and overwrite Content-Type if provided
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}

	resp, err := h.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if result == nil {
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, result)
}
