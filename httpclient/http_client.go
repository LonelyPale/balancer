package httpclient

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/bytom/blockcenter/balancer"
)

type HttpClient struct {
	Balancer balancer.Balancer
}

func New(opts balancer.Options) (*HttpClient, error) {
	loadBalancing, err := balancer.Manager.Balancer(&opts)
	if err != nil {
		return nil, err
	}

	return &HttpClient{
		Balancer: loadBalancing,
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
	resp, err := h.Balancer.Do(req)

	//Failure retry
	if err != nil || (resp != nil && resp.StatusCode >= 400) {
		for i := 0; i < 3; i++ {
			resp, err = h.Balancer.Do(req)
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
