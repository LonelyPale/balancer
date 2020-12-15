package websocket

import (
	"github.com/bytom/blockcenter/balancer"
	"github.com/bytom/blockcenter/service/websocket"
)

func NewClient(path string, processCh chan *websocket.WSResponse, opts balancer.Options) (*websocket.Client, error) {
	loadBalancing, err := balancer.Manager.Balancer(&opts)
	if err != nil {
		return nil, err
	}

	backend, err := loadBalancing.Pick()
	if err != nil {
		return nil, err
	}

	return websocket.NewClient(backend.URL, path, processCh), nil
}
