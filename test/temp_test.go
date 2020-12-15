package test

import (
	"testing"

	"github.com/bytom/blockcenter/config"
)

func TestConfig(t *testing.T) {
	filepath := "/Users/wyb/project/github/blockcenter/config_balancer.json"
	cfgs := config.NewConfigWithPath(filepath)

	t.Log(cfgs["btm"].Balancer)
}

func TestWatchConfig(t *testing.T) {
	filepath := "/Users/wyb/project/github/blockcenter/config_balancer.json"
	cfgs := config.NewConfigWithPath(filepath)

	t.Log(cfgs["btm"].Balancer)

	select {}
}

func TestAddress(t *testing.T) {
	address := "bm1q50u3z8empm5ke0g3ngl2t3sqtr6sd7cepd3z68"
	t.Log(address)
}
