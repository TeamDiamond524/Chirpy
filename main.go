package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
    fileserverHits atomic.Int32
}

func main() {
    var apiCfg apiConfig
    mux := http.NewServeMux()

    mux.HandleFunc("GET /api/healthz", ReadinessHandler)
    mux.HandleFunc("GET /admin/metrics", apiCfg.MetricsHandler)
    mux.HandleFunc("POST /admin/reset",   apiCfg.ResetHandler)
    mux.HandleFunc("POST /api/validate_chirp", ChirpValidationHandler)

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
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(200)
    w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, a.fileserverHits.Load())))
}

func (a *apiConfig) ResetHandler(w http.ResponseWriter, r *http.Request) {
    a.fileserverHits.Store(0)
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(200)
}

func ChirpValidationHandler(w http.ResponseWriter, r *http.Request) {
    type reqParameters struct {
        Body string `json:"body"`
    }

    decoder := json.NewDecoder(r.Body)

    clientInput := reqParameters{}
    if err := decoder.Decode(&clientInput); err != nil {
        fmt.Printf("Error decoding parameters: %v\n", err)
        RespondWithError(w, 400, "Something went wrong")
        return
    }

    if len(clientInput.Body) > 140 {
        RespondWithError(w, 400, "Chirp is too long")
        return
    }

    type resParameters struct {
        Valid bool `json:"valid"`
        CleanedBody string `json:"cleaned_body"`
    }

    RespondWithJSON(w, 200, resParameters{Valid: true, CleanedBody: CleanBody(clientInput.Body)})
}

func RespondWithError(w http.ResponseWriter, code int, errMsg string) {
    type errorParameters struct {
        Error string `json:"error"`
    }

    w.WriteHeader(code)

    resError := errorParameters{
        Error: errMsg,
    }

    resData, err := json.Marshal(resError)
    if err != nil {
        fmt.Printf("Error marshalling JSON: %s\n", err)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(resData)
}

func RespondWithJSON(w http.ResponseWriter, code int, payload any) {
    resData, err := json.Marshal(payload)
    if err != nil {
        fmt.Printf("Error marshalling JSON: %s\n", err)
        w.WriteHeader(400)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(resData)
}

func CleanBody(body string) string {
    banned_words := []string{"kerfuffle", "sharbert", "fornax"}
    words := strings.Split(body, " ")

    for i, word := range words {
        if slices.Contains(banned_words, strings.ToLower(word)) {
            words[i] = "****"
        }
    }
    
    return strings.Join(words, " ")
}
