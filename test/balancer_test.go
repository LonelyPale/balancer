package test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bytom/blockcenter/balancer"
	"github.com/bytom/blockcenter/balancer/httpclient"
	_ "github.com/bytom/blockcenter/balancer/round_robin" //Register the actual load algorithm used
)

func Test(t *testing.T) {
	opts := balancer.Options{
		Name:    "test",
		Type:    "RoundRobin",
		Timeout: 3,
		Doctor: balancer.DoctorOptions{
			Enable: true,
			Type:   "Default",
			Spec:   "*/2 * * * *",
		},
		Statistic: balancer.StatisticOptions{
			Enable: true,
			Port:   30000,
		},
		Urls: []string{"http://www.baidu.com", "https://www.baidu.com/home/skin/submit/activitylottery"},
	}

	client, err := httpclient.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		result := make(map[string]interface{})
		if err := client.Get("/", &result); err != nil {
		}
		t.Log(result)
	}
}

func TestBalancer(t *testing.T) {
	go testServer()
	time.Sleep(1 * time.Second)

	opts := balancer.Options{
		Name:    "test",
		Type:    "RoundRobin",
		Timeout: 3,
		Doctor: balancer.DoctorOptions{
			Enable: true,
			Type:   "Default",
			Spec:   "*/2 * * * *",
		},
		Statistic: balancer.StatisticOptions{
			Enable: true,
			Port:   30000,
		},
		Urls: []string{
			"http://localhost:10000/api1",
			"http://localhost:10000/api2",
			"http://localhost:10000/api3",
			"localhost:10000/api4",
			"localhost:10000/api5",
			"localhost:10000/api6",
		},
	}

	client, err := httpclient.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1200; i++ {
		result := make(map[string]interface{})
		if err := client.Get("", &result); err != nil {
		}
		//t.Log(result)
	}

	resp, err := http.Get("http://localhost:30000/balancer/statistic?name=test")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	result := make([]interface{}, 0)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}

	t.Log(result)
	assert.Equal(t, len(result), 6)
	for _, v := range result {
		m := v.(map[string]interface{})
		switch m["url"] {
		case "http://localhost:10000/api1":
			assert.Equal(t, m["alive"], true)
			assert.Equal(t, m["success"], float64(400))
		case "http://localhost:10000/api2":
			assert.Equal(t, m["alive"], false)
			assert.Equal(t, m["failure"], float64(100))
		case "http://localhost:10000/api3":
			assert.Equal(t, m["alive"], false)
			assert.Equal(t, m["failure"], float64(100))
		case "http://localhost:10000/api4":
			assert.Equal(t, m["alive"], true)
			assert.Equal(t, m["success"], float64(400))
		case "http://localhost:10000/api5":
			assert.Equal(t, m["alive"], true)
			assert.Equal(t, m["success"], float64(400))
		case "http://localhost:10000/api6":
			assert.Equal(t, m["alive"], false)
			assert.Equal(t, m["failure"], float64(100))
		}
	}

}
