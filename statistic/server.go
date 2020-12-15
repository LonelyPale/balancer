package statistic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytom/blockcenter/balancer"
)

func ServerAndRun(statistic *balancer.StatisticOptions) {
	mux := http.NewServeMux()
	mux.HandleFunc("/balancer/statistic", indexHandler)

	addr := ":" + strconv.Itoa(statistic.Port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	lb := balancer.Manager.Get(name)
	var body []byte

	if lb != nil {
		result := make([]interface{}, 0)

		lb.Backends().Range(func(index int, backend *balancer.Backend) bool {
			url := backend.URL
			alive := backend.State.Alive()
			success := backend.Statistic.Success()
			failure := backend.Statistic.Failure()
			content := make(map[string]interface{})
			content["url"] = url
			content["alive"] = alive
			content["success"] = success
			content["failure"] = failure
			result = append(result, content)
			return true
		})

		var err error
		body, err = json.Marshal(result)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		body = []byte("not found balancer " + name + "\n")
	}

	if _, err := w.Write(body); err != nil {
		fmt.Println(err)
	}
}
