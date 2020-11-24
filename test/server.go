package test

import (
	"net/http"
)

func testServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api1", api1)
	mux.HandleFunc("/api2", api2)
	mux.HandleFunc("/api3", api3)
	mux.HandleFunc("/api4", api4)
	mux.HandleFunc("/api5", api5)
	mux.HandleFunc("/api6", api6)

	if err := http.ListenAndServe(":10000", mux); err != nil {
		panic(err)
	}
}

func api1(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`{"msg":"ok-1"}`))
}

func api2(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(500)
	_, _ = w.Write([]byte(`{"msg":"fail-2"}`))
}

func api3(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(500)
	_, _ = w.Write([]byte(`{"msg":"fail-3"}`))
}

func api4(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`{"msg":"ok-4"}`))
}

func api5(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`{"msg":"ok-5"}`))
}

func api6(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(500)
	_, _ = w.Write([]byte(`{"msg":"fail-6"}`))
}
