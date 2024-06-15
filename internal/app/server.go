package app

import (
	"log"
	"net/http"
)

type server struct {
    mux *http.ServeMux
}

func newServer() *server {
    return &server{
        mux: http.NewServeMux(),
    }
}

func addRoutes(srv *server) {
    srv.mux.HandleFunc("GET /callback", handleAuth())
}

func handleAuth() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log.Print("yerr")
    }
}

func StartServer() error {
    srv := newServer()

    addRoutes(srv)

    return http.ListenAndServe(":3000", srv.mux)
}
