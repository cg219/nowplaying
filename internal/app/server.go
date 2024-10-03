package app

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cg219/nowplaying/internal/database"
	"github.com/cg219/nowplaying/pkg/argon2id"
	"github.com/cg219/nowplaying/pkg/webtoken"
	"github.com/golang-jwt/jwt/v5"
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
    srv.mux.Handle("GET /test/x", srv.handle(srv.UserOnly, srv.Test))
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

func (s *Server) Test(w http.ResponseWriter, r *http.Request) error {
    type TestResp struct {
        SuccessResp
        Value string `json:"value"`
    }

    resp := TestResp{ Value: r.Context().Value("username").(string)}
    resp.Success = true
    encode[TestResp](w, http.StatusOK, resp)
    return nil
}

func (s *Server) UserOnly(w http.ResponseWriter, r *http.Request) error {
    accessTokenExpired := true
    refreshTokenExpired := true
    cookie, err := r.Cookie("nowplaying")
    if err != nil {
        s.log.Error("Cookie Retrieval", "cookie", "nowplaying", "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    value, err := base64.StdEncoding.DecodeString(cookie.Value)
    if err != nil {
        s.log.Error("Base64 Decoding", "cookie", cookie.Value, "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    var cookieValue webtoken.CookieAuthValue
    err = json.Unmarshal(value, &cookieValue)
    if err != nil {
        s.log.Error("Invalid Cookie Value", "cookie", cookie.Value, "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    accessToken, err := webtoken.GetParsedJWT(cookieValue.AccessToken, "notsecure")
    if err != nil {
        fmt.Println()

        if !strings.Contains(err.Error(), jwt.ErrTokenExpired.Error()) {
            s.log.Error("Invalid AccessToken", "accessToken", cookieValue.AccessToken, "method", "UserOnly", "request", r, "error", err.Error())
            return fmt.Errorf(AUTH_ERROR)
        }
    } else {
        accessTokenExpired = false
    }

    refreshToken, err := webtoken.GetParsedJWT(cookieValue.RefreshToken, "notsecure")
    if err != nil {
        if !strings.Contains(err.Error(), jwt.ErrTokenExpired.Error()) {
            s.log.Error("Invalid RefreshToken", "refreshToken", cookieValue.RefreshToken, "method", "UserOnly", "request", r, "error", err.Error())
            return fmt.Errorf(AUTH_ERROR)
        }
    } else {
        refreshTokenExpired = false
    }

    rfs, err := refreshToken.Claims.GetSubject()
    if err != nil {
        s.log.Error("Invalid RefreshToken", "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    var rf webtoken.Subject
    err = json.Unmarshal([]byte(rfs), &rf)
    if err != nil {
        s.log.Error("Invalid RefreshToken", "refreshToken", rfs, "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    if refreshTokenExpired {
        s.log.Error("Expired RefreshToken", "refreshToken", cookieValue.RefreshToken, "method", "UserOnly", "request", r)
        s.authCfg.database.InvalidateUserSession(r.Context(), database.InvalidateUserSessionParams{
            Accesstoken: cookieValue.AccessToken,
            Refreshtoken: rf.Value,
        })
        return fmt.Errorf(AUTH_ERROR)
    }

    _, err = s.authCfg.database.GetUserSession(r.Context(), database.GetUserSessionParams{
        Accesstoken: cookieValue.AccessToken,
        Refreshtoken: rf.Value,
    })
    if err != nil {
        s.log.Error("Retreiving User Session", "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    us, err := accessToken.Claims.GetSubject()
    if err != nil {
        s.log.Error("Invalid AccessToken", "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    var username webtoken.Subject
    err = json.Unmarshal([]byte(us), &username)
    if err != nil {
        s.log.Error("Invalid AccessToken", "accessToken", us, "method", "UserOnly", "request", r, "error", err.Error())
        return fmt.Errorf(AUTH_ERROR)
    }

    if accessTokenExpired {
        s.log.Error("Expired AccessToken", "accessToken", cookieValue.AccessToken, "method", "UserOnly", "request", r )
        s.authCfg.database.InvalidateUserSession(r.Context(), database.InvalidateUserSessionParams{
            Accesstoken: cookieValue.AccessToken,
            Refreshtoken: rf.Value,
        })
        s.RefreshAccessToken(r.Context(), rf.Value, username.Value, w)
    }

    ctx := context.WithValue(r.Context(), "username", username.Value)
    updatedRequest := r.WithContext(ctx)

    *r = *updatedRequest
    return nil
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

    accessToken := webtoken.NewToken("accessToken", body.Username, "notsecure", time.Now().Add(time.Hour * 1))
    refreshToken := webtoken.NewToken("refreshToken", webtoken.GenerateRefreshString(), "notsecure", time.Now().Add(time.Hour * 24 * 30))
    accessToken.Create("nowplaying")
    refreshToken.Create("nowplaying")
    cookieValue := webtoken.CookieAuthValue{ AccessToken: accessToken.Value(), RefreshToken: refreshToken.Value() }
    cookie := webtoken.NewAuthCookie("nowplaying", "/", cookieValue, int(time.Hour * 24 * 30))

    s.authCfg.database.SaveUserSession(r.Context(), database.SaveUserSessionParams{
        Accesstoken: accessToken.Value(),
        Refreshtoken: refreshToken.Subject(),
    })

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

    accessToken := webtoken.NewToken("accessToken", body.Username, "notsecure", time.Now().Add(time.Hour* 1))
    refreshToken := webtoken.NewToken("refreshToken", webtoken.GenerateRefreshString(), "notsecure", time.Now().Add(time.Hour * 24 * 30))
    accessToken.Create("nowplaying")
    refreshToken.Create("nowplaying")
    cookieValue := webtoken.CookieAuthValue{ AccessToken: accessToken.Value(), RefreshToken: refreshToken.Value() }
    cookie := webtoken.NewAuthCookie("nowplaying", "/", cookieValue, int(time.Hour * 24 * 30))

    s.authCfg.database.SaveUserSession(r.Context(), database.SaveUserSessionParams{
        Accesstoken: accessToken.Value(),
        Refreshtoken: refreshToken.Subject(),
    })

    http.SetCookie(w, &cookie)
    encode[SuccessResp](w, http.StatusOK, SuccessResp{ Success: true })
    s.log.Info("Login Body", body)
    return nil
}

func (s* Server) RefreshAccessToken(ctx context.Context, refresh, username string, w http.ResponseWriter) {
    accessToken := webtoken.NewToken("accessToken", username, "notsecure", time.Now().Add(time.Hour * 1))
    refreshToken := webtoken.NewToken("refreshToken", refresh, "notsecure", time.Now().Add(time.Hour * 24 * 30))
    accessToken.Create("nowplaying")
    refreshToken.Create("nowplaying")
    cookieValue := webtoken.CookieAuthValue{ AccessToken: accessToken.Value(), RefreshToken: refreshToken.Value() }
    cookie := webtoken.NewAuthCookie("nowplaying", "/", cookieValue, int(time.Hour * 24 * 30))

    s.authCfg.database.SaveUserSession(ctx, database.SaveUserSessionParams{
        Accesstoken: accessToken.Value(),
        Refreshtoken: refreshToken.Subject(),
    })

    http.SetCookie(w, &cookie)
    s.log.Info("Refresh User Tokens", "username", username)
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
