package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cg219/nowplaying/internal/database"
	"github.com/cg219/nowplaying/pkg/argon2id"
	"github.com/cg219/nowplaying/pkg/webtoken"
)

type Server struct {
    mux *http.ServeMux
    authCfg *AuthCfg
    log *slog.Logger
    hasher *argon2id.Argon2id
}

type SuccessResp struct {
    Success bool `json:"success"`
}

type ResponseError struct {
    Code int `json:"code"`
    Success bool `json:"success"`
    Messaage string `json:"message"`
}

type CandlerFunc func(w http.ResponseWriter, r *http.Request) error

const (
    INTERNAL_ERROR = "Internal Server Error"
    AUTH_ERROR = "Authentication Error"
    USERNAME_EXISTS_ERROR = "Username Exists Error"
)
const (
    CODE_USER_EXISTS = iota
    AUTH_FAIL
    AUTH_NOT_ALLOWED
    INTERNAL_SERVER_ERROR
)

func NewServer(cfg *AuthCfg) *Server {
    return &Server{
        mux: http.NewServeMux(),
        authCfg: cfg,
        log: slog.New(slog.NewTextHandler(os.Stderr, nil)),
        hasher: argon2id.NewArgon2id(16 * 1024, 2, 1, 16, 32),
    }
}

func encode[T any](w http.ResponseWriter, status int, v T) error {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(v); err != nil {
        return fmt.Errorf("encoding: %w", err)
    }

    return nil
}

func decode[T any](r *http.Request) (T, error) {
    var v T
    if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
        return v, fmt.Errorf("decoding: %w", err)
    }

    return v, nil
}

func addRoutes(srv *Server) {
    srv.mux.Handle("GET /auth/spotify-redirect", srv.handle(srv.SpotifyRedirect))
    srv.mux.Handle("POST /auth/register", srv.handle(srv.Register))
    srv.mux.Handle("POST /auth/login", srv.handle(srv.Login))
    srv.mux.Handle("POST /auth/refresh", srv.handle(srv.RefreshAccessToken))
}

func return500(w http.ResponseWriter) {
    encode[ResponseError](w, 500, ResponseError{ Success: false, Messaage: INTERNAL_ERROR, Code: INTERNAL_SERVER_ERROR })
}

func (s *Server) handle(h ...CandlerFunc) http.Handler {

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        for _, currentHandler := range h {
            if err := currentHandler(w, r); err != nil {
                switch err.Error() {
                case USERNAME_EXISTS_ERROR:
                    if err := encode[ResponseError](w, 409, ResponseError{ Success: false, Messaage: "Username Taken", Code: CODE_USER_EXISTS }); err != nil {
                       return500(w)
                    }
                    return

                case AUTH_ERROR:
                    if err := encode[ResponseError](w, 404, ResponseError{ Success: false, Messaage: "Username/Password Incorrect", Code: AUTH_FAIL }); err != nil {
                        return500(w)
                    }
                    return
                    
                case INTERNAL_ERROR:
                    return500(w)
                    return
                }

                s.log.Error("Uncaught Error", err)
            }
        }
    })
}

func (h CandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if err := h(w, r); err != nil {
        fmt.Println("OOPS")
    }
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) error {
    type RegisterBody struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }

    body, err := decode[RegisterBody](r)
    if err != nil {
        return err
    }

    existingUser, err := s.authCfg.database.GetUser(r.Context(), body.Username)
    if err != nil && err != sql.ErrNoRows {
        s.log.Error("sql err: %w", err)
        return fmt.Errorf(INTERNAL_ERROR)
    }

    if existingUser.Username != "" {
        return fmt.Errorf(USERNAME_EXISTS_ERROR)
    }

    hashPass, err := s.hasher.EncodeFromString(body.Password)
    if err != nil {
        s.log.Error("Encoding Password", "password", body.Password)
        return fmt.Errorf(INTERNAL_ERROR)
    }

    err = s.authCfg.database.SaveUser(r.Context(), database.SaveUserParams{
        Username: body.Username,
        Password: hashPass,
    })

    if err != nil {
        s.log.Error("Saving New User", "username", body.Username, "err", err)
        return fmt.Errorf(INTERNAL_ERROR)
    }

    token := webtoken.NewToken("nowplaying-au", body.Username, "notsecure", time.Now().Add(time.Hour * 1))
    cookie := webtoken.NewCookie("nowplaying", token.Value(), "/", int(time.Hour * 24 * 30))

    http.SetCookie(w, &cookie)
    encode[SuccessResp](w, http.StatusOK, SuccessResp{ Success: true })
    s.log.Info("Register Body", body)
    return nil
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) error {
    type LoginBody struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }

    body, err := decode[LoginBody](r)
    if err != nil {
        return err
    }

    existingUser, err := s.authCfg.database.GetUserWithPassword(r.Context(), body.Username)
    if err != nil {
        if err == sql.ErrNoRows {
            return fmt.Errorf(AUTH_ERROR)
        }

        s.log.Error("sql err: %w", err)
        return fmt.Errorf(INTERNAL_ERROR)
    }

    if existingUser.Username == "" {
        return  fmt.Errorf(AUTH_ERROR) 
    }

    correct, _ := s.hasher.Compare(body.Password, existingUser.Password.(string))
    if !correct {
        s.log.Info("Password Mismatch", "password", body.Password)
        return fmt.Errorf(AUTH_ERROR)
    }

    token := webtoken.NewToken("nowplaying-au", body.Username, "notsecure", time.Now().Add(time.Hour * 1))
    cookie := webtoken.NewCookie("nowplaying", token.Value(), "/", int(time.Hour * 24 * 30))

    http.SetCookie(w, &cookie)
    encode[SuccessResp](w, http.StatusOK, SuccessResp{ Success: true })
    s.log.Info("Login Body", body)
    return nil
}

func (s *Server) RefreshAccessToken(w http.ResponseWriter, r *http.Request) error {
    return nil
}

func (s *Server) SpotifyRedirect(w http.ResponseWriter, r *http.Request) error {
    s.authCfg.SpotifySession.SetAuthCode(r.URL.Query().Get("code"))
    s.authCfg.SpotifySession.GetSpotifyTokens(r.Context())
    fmt.Println(r.URL.Query())
    return nil
}

func StartServer(cfg *AuthCfg) error {
    srv := NewServer(cfg)

    addRoutes(srv)

    return http.ListenAndServe("localhost:3006", srv.mux)
}
