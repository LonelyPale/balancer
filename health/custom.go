package health

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/lonelypale/balancer"
)

func BytomPing(backend *balancer.Backend) error {
	url := balancer.URLJoin(backend.URL, "/net-info")
	result := make(map[string]interface{})

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, result); err != nil {
		return err
	}

	return nil
}
