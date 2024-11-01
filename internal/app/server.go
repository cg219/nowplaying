package app

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	// _ "net/http/pprof"
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
    authCfg *AppCfg
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
    GOTO_NEXT_HANDLER_ERROR = "Redirect Error"
    REDIRECT_ERROR = "Intentional Redirect Error"
)
const (
    CODE_USER_EXISTS = iota
    AUTH_FAIL
    AUTH_NOT_ALLOWED
    INTERNAL_SERVER_ERROR
)

func NewServer(cfg *AppCfg) *Server {
    return &Server{
        mux: http.NewServeMux(),
        authCfg: cfg,
        log: slog.New(slog.NewTextHandler(os.Stderr, nil)),
        hasher: argon2id.NewArgon2id(16 * 1024, 2, 1, 16, 32),
    }
}

func addRoutes(srv *Server) {
    srv.mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusNotFound)
    })
    srv.mux.Handle("GET /", srv.handle(srv.RedirectAuthenticated("/settings", true), srv.getLoginPage))
    srv.mux.Handle("POST /api/login", srv.handle(srv.LogUserIn))
    srv.mux.Handle("GET /api/last-scrobble", srv.handle(srv.UserOnly, srv.GetLastScrobble))
    srv.mux.Handle("POST /api/spotify", srv.handle(srv.UserOnly, srv.AddSpotify))
    srv.mux.Handle("DELETE /api/spotify", srv.handle(srv.UserOnly, srv.RemoveSpotify))
    srv.mux.Handle("GET /auth/spotify-redirect", srv.handle(srv.SpotifyRedirect))
    srv.mux.Handle("GET /auth/x-redirect", srv.handle(srv.TwitterRedirect))
    srv.mux.Handle("POST /auth/register", srv.handle(srv.Register))
    srv.mux.Handle("POST /auth/login", srv.handle(srv.Login))
    srv.mux.Handle("GET /test/x", srv.handle(srv.UserOnly, srv.Test))
    srv.mux.Handle("GET /me", srv.handle(srv.RedirectAuthenticated("/", false), srv.getUserPage))
    srv.mux.Handle("GET /settings", srv.handle(srv.RedirectAuthenticated("/", false), srv.getSettingsPage))
}

func (h CandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if err := h(w, r); err != nil {
        fmt.Println("OOPS")
    }
}

func (s *Server) getLoginPage(w http.ResponseWriter, r *http.Request) error {
    tmpl := template.Must(template.ParseFiles("templates/pages/auth.html"))
    tmpl.Execute(w, nil)
    return nil
}

func (s *Server) getUserPage(w http.ResponseWriter, r *http.Request) error {
    tmpl := template.Must(template.ParseFiles("templates/pages/user.html"))
    tmpl.Execute(w, nil)
    return nil
}

func (s *Server) getSettingsPage(w http.ResponseWriter, r *http.Request) error {
    user, err := s.authCfg.database.GetUser(r.Context(), r.Context().Value("username").(string))
    if err != nil && err != sql.ErrNoRows {
        return err
    }

    sessions, err := s.authCfg.database.GetUserMusicSessions(r.Context(), user.ID)
    if err != nil && err != sql.ErrNoRows {
        return err
    }

    spotify, err := s.authCfg.database.GetSpotifySession(r.Context(), r.Context().Value("username").(string))
    if err != nil && err != sql.ErrNoRows {
        return err
    }

    twitter, err := s.authCfg.database.GetTwitterSession(r.Context(), r.Context().Value("username").(string))
    if err != nil && err != sql.ErrNoRows {
        return err
    }

    page := &struct{
        SpotifyTrack string
        SpotifyAuthURL string
        SpotifyOn bool
        TwitterOn bool
        TwitterAuthURL string
    }{}

    if spotify.SpotifyAccessToken.Valid && spotify.SpotifyRefreshToken.Valid {
        page.SpotifyOn = true
    } else {
        page.SpotifyAuthURL = GetSpotifyAuthURL(r.Context(), user.Username, SpotifyConfig{
            Id: s.authCfg.config.Spotify.Id,
            Redirect: s.authCfg.config.Spotify.Redirect,
            Secret: s.authCfg.config.Spotify.Secret,
        }, s.authCfg.database)
    }

    for _, v := range sessions {
        if strings.EqualFold(v.Type, "spotify") {
            if v.Active == 1 {
                page.SpotifyTrack = "checked"
            }
        }
    }

    if twitter.TwitterOauthToken.Valid && twitter.TwitterOauthSecret.Valid {
        page.TwitterOn = true
    } else {
        page.TwitterAuthURL = GetAuthURL(s.authCfg.ctx, s.authCfg.TwitterOAuth, s.authCfg.database, user.Username)
    }

    tmpl := template.Must(template.ParseFiles("templates/pages/settings.html"))
    tmpl.Execute(w, page)
    return nil
}

func (s *Server) setTokens(w http.ResponseWriter, r *http.Request, username string) {
    accessToken := webtoken.NewToken("accessToken", username, "notsecure", time.Now().Add(time.Hour * 1))
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
}

func (s *Server) authenticateRequest(r *http.Request, username string) {
    ctx := context.WithValue(r.Context(), "username", username)
    updatedRequest := r.WithContext(ctx)

    *r = *updatedRequest
}

func (s *Server) getAuthGookie(r *http.Request) (string, string) {
    cookie, err := r.Cookie("nowplaying")
    if err != nil {
        s.log.Error("Cookie Retrieval", "cookie", "nowplaying", "method", "UserOnly", "request", r, "error", err.Error())
        return "", ""
    }

    value, err := base64.StdEncoding.DecodeString(cookie.Value)
    if err != nil {
        s.log.Error("Base64 Decoding", "cookie", cookie.Value, "method", "UserOnly", "request", r, "error", err.Error())
        return "", ""
    }

    var cookieValue webtoken.CookieAuthValue
    err = json.Unmarshal(value, &cookieValue)
    if err != nil {
        s.log.Error("Invalid Cookie Value", "cookie", cookie.Value, "method", "UserOnly", "request", r, "error", err.Error())
        return "", ""
    }

    return cookieValue.AccessToken, cookieValue.RefreshToken
}

func (s *Server) login(ctx context.Context, username string, password string) bool {
    existingUser, err := s.authCfg.database.GetUserWithPassword(ctx, username)
    if err != nil {
        if err == sql.ErrNoRows {
            return false
        }

        s.log.Error("sql err: %w", err)
        return false
    }

    if existingUser.Username == "" {
        return false 
    }

    correct, _ := s.hasher.Compare(password, existingUser.Password.(string))
    if !correct {
        s.log.Info("Password Mismatch", "password", password)
        return false
    }

    return true
}

func (s* Server) refreshAccessToken(ctx context.Context, refreshExpire int64, refreshTokenString, refreshValue, username string, w http.ResponseWriter) {
    accessToken := webtoken.NewToken("accessToken", username, "notsecure", time.Now().Add(time.Hour * 1))
    accessToken.Create("nowplaying")
    cookieValue := webtoken.CookieAuthValue{ AccessToken: accessToken.Value(), RefreshToken: refreshTokenString }
    cookie := webtoken.NewAuthCookie("nowplaying", "/", cookieValue, int(refreshExpire))

    s.authCfg.database.SaveUserSession(ctx, database.SaveUserSessionParams{
        Accesstoken: accessToken.Value(),
        Refreshtoken: refreshValue,
    })

    http.SetCookie(w, &cookie)
    s.log.Info("Refresh User Tokens", "username", username)
}

func (s *Server) isAuthenticated(ctx context.Context, ats, rts string) (bool, string, func(http.ResponseWriter)) {
    accessTokenExpired := true
    refreshTokenExpired := true
    accessToken, err := webtoken.GetParsedJWT(ats, "notsecure")
    if err != nil {
        fmt.Println()

        if !strings.Contains(err.Error(), jwt.ErrTokenExpired.Error()) {
            s.log.Error("Invalid AccessToken", "accessToken", ats, "method", "IsAuthenticated", "error", err.Error())
            return false, "", nil
        }
    } else {
        accessTokenExpired = false
    }

    refreshToken, err := webtoken.GetParsedJWT(rts, "notsecure")
    if err != nil {
        if !strings.Contains(err.Error(), jwt.ErrTokenExpired.Error()) {
            s.log.Error("Invalid RefreshToken", "refreshToken", rts, "method", "isAuthenticated", "error", err.Error())
            return false, "", nil
        }
    } else {
        refreshTokenExpired = false
    }

    rfs, err := refreshToken.Claims.GetSubject()
    if err != nil {
        s.log.Error("Invalid RefreshToken", "method", "isAuthenticated", "error", err.Error())
        return false, "", nil
    }

    var rf webtoken.Subject
    err = json.Unmarshal([]byte(rfs), &rf)
    if err != nil {
        s.log.Error("Invalid RefreshToken", "refreshToken", rfs, "method", "isAuthenticated", "error", err.Error())
        return false, "", nil
    }

    if refreshTokenExpired {
        s.log.Error("Expired RefreshToken", "refreshToken", rts, "method", "isAuthenticated")
        s.authCfg.database.InvalidateUserSession(ctx, database.InvalidateUserSessionParams{
            Accesstoken: ats,
            Refreshtoken: rf.Value,
        })
        return false, "", nil
    }

    _, err = s.authCfg.database.GetUserSession(ctx, database.GetUserSessionParams{
        Accesstoken: ats,
        Refreshtoken: rf.Value,
    })
    if err != nil {
        s.log.Error("Retreiving User Session", "method", "isAuthenticated", "error", err.Error())
        return false, "", nil
    }

    us, err := accessToken.Claims.GetSubject()
    if err != nil {
        s.log.Error("Invalid AccessToken", "method", "isAuthenticated", "error", err.Error())
        return false, "", nil
    }

    var username webtoken.Subject
    err = json.Unmarshal([]byte(us), &username)
    if err != nil {
        s.log.Error("Invalid AccessToken", "accessToken", us, "method", "isAuthenticated", "error", err.Error())
        return false, "", nil
    }

    if accessTokenExpired {
        s.log.Error("Expired AccessToken", "accessToken", ats, "method", "isAuthenticated")
        s.authCfg.database.InvalidateUserSession(ctx, database.InvalidateUserSessionParams{
            Accesstoken: ats,
            Refreshtoken: rf.Value,
        })

        expiresAt, _ := refreshToken.Claims.GetExpirationTime()

        return false, username.Value, func(w http.ResponseWriter) {
            s.refreshAccessToken(ctx, expiresAt.Unix(), rts, rf.Value, username.Value, w)
        }
    }

    return true, username.Value, nil
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

    s.setTokens(w, r, body.Username)
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

    if !s.login(r.Context(), body.Username, body.Password) {
        return fmt.Errorf(AUTH_ERROR)
    }

    s.setTokens(w, r, body.Username)
    encode[SuccessResp](w, http.StatusOK, SuccessResp{ Success: true })
    s.log.Info("Login Body", body)
    return nil
}

func StartServer(cfg *AppCfg) error {
    srv := NewServer(cfg)

    addRoutes(srv)

    return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT")), srv.mux)
}
