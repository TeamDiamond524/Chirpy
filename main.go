package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/TeamDiamond524/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
    fileserverHits atomic.Int32
    dbQueires *database.Queries
    platform string
}

func main() {
    fmt.Printf("Starting server\n")
    godotenv.Load()
    dbURL := os.Getenv("DB_URL")

    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        fmt.Printf("Failed to connect with database, %s", err)
        return
    }

    var apiCfg apiConfig
    mux := http.NewServeMux()

    fmt.Printf("Setting up database connection\n")
    apiCfg.dbQueires = database.New(db)
    apiCfg.platform = os.Getenv("PLATFORM")

    fmt.Printf("Setting up handlers\n")
    mux.HandleFunc("GET /api/healthz", ReadinessHandler)
    mux.HandleFunc("GET /admin/metrics", apiCfg.MetricsHandler)
    mux.HandleFunc("POST /admin/reset",   apiCfg.ResetHandler)
    mux.HandleFunc("POST /api/chirps", apiCfg.ChirpsHandler)
    mux.HandleFunc("POST /api/users", apiCfg.CreateUserHandler)

    mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))

    server := http.Server{
        Handler: mux,
        Addr: ":8080",
    }

    fmt.Printf("Server is set up\n")
    if err := server.ListenAndServe(); err != nil {
        fmt.Printf("Error while running server: %s", err)
    }
    fmt.Printf("Closing server\n")
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
    w.Write(fmt.Appendf([]byte{}, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, a.fileserverHits.Load()))
}

func (a *apiConfig) ResetHandler(w http.ResponseWriter, r *http.Request) {
    if a.platform != "dev" {
        w.WriteHeader(403)
        return
    }

    a.fileserverHits.Store(0)
    a.dbQueires.ResetUsers(context.Background())
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(200)
}

func (a *apiConfig) ChirpsHandler(w http.ResponseWriter, r *http.Request) {
    decoder := json.NewDecoder(r.Body)

    clientInput := database.CreateChirpParams{}
    if err := decoder.Decode(&clientInput); err != nil {
        fmt.Printf("Error decoding parameters: %v\n", err)
        RespondWithError(w, 400, "Something went wrong")
        return
    }

    if len(clientInput.Body) > 140 {
        RespondWithError(w, 400, "Chirp is too long")
        return
    }

    chirp, err := a.dbQueires.CreateChirp(context.Background(), clientInput)
    if err != nil {
        fmt.Printf("Error while creating new chirp: %s\n", err)
        RespondWithError(w, 400, "Something went wrong")
        return
    }

    RespondWithJSON(w, 201, chirp)
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

func (a *apiConfig) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    type userParam struct {
        Email string `json:"email"`
    }

    var email userParam
    decoder := json.NewDecoder(r.Body)

    if err := decoder.Decode(&email); err != nil {
        fmt.Printf("Error while decoding json: %s\n", err)
        w.WriteHeader(400)
        return
    }

    user, err := a.dbQueires.CreateUser(context.Background(), email.Email)
    if err != nil {
        fmt.Printf("Errorn while creating user: %s\n", err)
        w.WriteHeader(400)
        return
    }

    userData, err := json.Marshal(user)
    if err != nil {
        fmt.Printf("Error while marshaling data: %s\n", err)
        w.WriteHeader(400)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(201)
    w.Write(userData)
}
