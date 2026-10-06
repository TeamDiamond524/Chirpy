package main

import (
	"net/http"
)

func main() {
    mux := http.NewServeMux()

    mux.HandleFunc("/healthz", ReadinessHandler)

    mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))

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
