package management

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"
)

type handler = func(w http.ResponseWriter, r *http.Request)

var mux = make(map[string]func(w http.ResponseWriter, r *http.Request))

func StartServer() *http.Server {
	server := http.Server{Addr: ":1337", Handler: &jsonHandler{}, ReadTimeout: 5 * time.Second}
	mux["/metrics"] = metricsEndpoint
	mux["/instrumentation"] = instrumentationProvidersEndpoint
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

func RefreshInstrumentationEndpoints() {
	for _, p := range instrumentationProviders {
		ep := fmt.Sprintf("/instrumentation/%s", p.Name())
		mux[ep] = makeProviderListEndpoint(p.Name())
	}
}

type MetricResponse struct {
	GoStats  GoStats
	Counters map[string]int64
}

type GoStats struct {
	Memory runtime.MemStats
}

func writeJson(obj any, w http.ResponseWriter) {
	jsonRes, err := json.Marshal(obj)

	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, string(jsonRes))
}

func metricsEndpoint(w http.ResponseWriter, r *http.Request) {
	var memoryStats runtime.MemStats
	runtime.ReadMemStats(&memoryStats)

	goStats := GoStats{Memory: memoryStats}
	metricResponse := MetricResponse{
		GoStats:  goStats,
		Counters: GetCounters(),
	}

	writeJson(metricResponse, w)
}

type ProviderResponse struct {
	Providers []string
}

func instrumentationProvidersEndpoint(w http.ResponseWriter, r *http.Request) {
	res := make([]string, 0)

	for _, v := range GetProviders() {
		res = append(res, v.Name())
	}

	writeJson(ProviderResponse{Providers: res}, w)
}

type ListResponse struct {
	Items []InstrumentationEntry
}

func makeProviderListEndpoint(name string) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		providerList(w, name)
	}
}

func providerList(w http.ResponseWriter, name string) {
	provider, ok := GetProvider(name);
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, fmt.Sprintf("Provider '%s' not found.", name))
		return
	}

	writeJson(ListResponse{Items: provider.List()}, w)
}
