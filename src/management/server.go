package management

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"
)

var mux map[string]func(w http.ResponseWriter, r *http.Request)

func StartServer() *http.Server {
	server := http.Server{Addr: ":1337", Handler: &jsonHandler{}, ReadTimeout: 5 * time.Second}
	mux = make(map[string]func(w http.ResponseWriter, r *http.Request))
	mux["/metrics"] = Metrics
	go server.ListenAndServe()
	return &server
}

type jsonHandler struct{}

func (*jsonHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := mux[r.URL.String()]; ok {
		w.Header().Add("Content-Type", "text/json")
		h(w, r)
		return
	}
	io.WriteString(w, "Unmapped URL: "+r.URL.String())
}

type MetricResponse struct {
	GoStats  GoStats
	Counters map[string]int64
}

type GoStats struct {
	Memory runtime.MemStats
}

func Metrics(w http.ResponseWriter, r *http.Request) {
	var memoryStats runtime.MemStats
	runtime.ReadMemStats(&memoryStats)

	goStats := GoStats{Memory: memoryStats}
	metricResponse := MetricResponse{
		GoStats:  goStats,
		Counters: GetCounters(),
	}

	jsonRes, err := json.Marshal(metricResponse)

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, string(jsonRes))
}
