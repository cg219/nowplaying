package app

import (
	"log"
	"net/http"
)

type server struct {
    mux *http.ServeMux
    authCfg *AuthCfg
}

func newServer(cfg *AuthCfg) *server {
    return &server{
        mux: http.NewServeMux(),
        authCfg: cfg,
    }
}

func addRoutes(srv *server) {
    srv.mux.HandleFunc("GET /redirect", handleAuth(srv.authCfg))
}

func handleAuth(cfg *AuthCfg) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        cfg.GetSpotifyTokens(r.URL.Query().Get("code"))
        log.Println(r.URL.Query())
        log.Println("yerr")
    }
}

func StartServer(cfg *AuthCfg) error {
    srv := newServer(cfg)

    addRoutes(srv)

    return http.ListenAndServe("localhost:3006", srv.mux)
}
