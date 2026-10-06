package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
    fileserverHits atomic.Int32
}

func main() {
    var apiCfg apiConfig
    mux := http.NewServeMux()

    mux.HandleFunc("/healthz", ReadinessHandler)
    mux.HandleFunc("/metrics", apiCfg.MetricsHandler)
    mux.HandleFunc("/reset",   apiCfg.ResetHandler)

    mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))

    server := http.Server{
        Handler: mux,
        Addr: ":8080",
    }

    server.ListenAndServe()
}

func ReadinessHandler(res http.ResponseWriter, req *http.Request) {
    //Set header type
    res.Header().Set("Content-Type", "text/plain; charset=utf-8")

    //Set custom status code
    res.WriteHeader(200)

    //Write body
    res.Write([]byte("OK"))
}

func (a *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        a.fileserverHits.Add(1)
        next.ServeHTTP(w, r)
    })
}

func (a *apiConfig) MetricsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(200)
    w.Write(fmt.Appendf([]byte{}, "Hits: %d", a.fileserverHits.Load()))
}

func (a *apiConfig) ResetHandler(w http.ResponseWriter, r *http.Request) {
    a.fileserverHits.Store(0)
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(200)
}
