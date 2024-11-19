package app

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/cg219/nowplaying/internal/database"
	"github.com/dghubble/oauth1"
)

func (s *Server) ResetPassword(w http.ResponseWriter, r *http.Request) error {
    resettimer := time.Now().Unix()
    err := r.ParseForm()

    if err != nil {
        s.log.Error("parsing form", "err", err)
    }

    pass := r.FormValue("password")
    passconfirm := r.FormValue("password-confirm")
    reset := r.FormValue("reset")

    if !strings.EqualFold(pass, passconfirm) {
        return fmt.Errorf(AUTH_ERROR)
    }

    hashPass, _ := s.hasher.EncodeFromString(pass)

    s.authCfg.database.ResetPassword(r.Context(), database.ResetPasswordParams{
        Reset: sql.NullString{ String: reset, Valid: true },
        ResetTime: sql.NullInt64{ Int64: resettimer, Valid: true },
        Password: hashPass,
    })

    http.Redirect(w, r, "/", http.StatusSeeOther)

    return nil
}

func (s *Server) ForgotPassword(w http.ResponseWriter, r *http.Request) error {
    resettimer := time.Now().Add(time.Minute * 15).Unix()
    resetbytes := make([]byte, 32)
    rand.Read(resetbytes)
    reset := base64.URLEncoding.EncodeToString(resetbytes)[:16]
    err := r.ParseForm()

    if err != nil {
        s.log.Error("parsing form", "err", err)
    }

    username := r.FormValue("username")

    err = s.authCfg.database.SetPasswordReset(r.Context(), database.SetPasswordResetParams{
        Reset: sql.NullString{ String: reset, Valid: true },
        ResetTime: sql.NullInt64{ Int64: resettimer, Valid: true },
        Username: username,
    })
    
    if err != nil {
        s.log.Error("resetting pass", "err", err)
    }

    s.log.Info("Reset Link:", "url", fmt.Sprintf("http://localhost:%s/reset/%s", "3006", reset))

    return nil
}

func (s *Server) AddSpotify(w http.ResponseWriter, r *http.Request) error {
    user, err := s.authCfg.database.GetUser(r.Context(), r.Context().Value("username").(string))
    if err != nil && err != sql.ErrNoRows {
       return err
    }

    sessions, err := s.authCfg.database.GetUserMusicSessions(r.Context(), user.ID)
    if err != nil && err != sql.ErrNoRows {
       return err
    }

    for _, v := range sessions {
        if strings.EqualFold(v.Type, "spotify") {
            if v.Active == 1 {
                return nil
            }

            err := s.authCfg.database.ActivateMusicSession(r.Context(), v.ID)
            if err != nil {
                s.log.Error("Activating Music Session", "Session ID", v.ID, "error", err)
                return fmt.Errorf(INTERNAL_ERROR)
            }

            s.authCfg.haveNewSessions = true
            return nil
        }
    }

    spotify := NewSpotify(r.Context().Value("username").(string), SpotifyConfig(s.authCfg.config.Spotify), s.authCfg.database)
    data := base64.StdEncoding.EncodeToString(spotify.Encode())
    err = s.authCfg.database.SaveMusicSession(r.Context(), database.SaveMusicSessionParams{
        Data: data,
        Type: "spotify",
        Uid: user.ID,
        Active: 1,
    })

    if err != nil {
        s.log.Error("Saving Music Session", "data", spotify, "error", err)
        return fmt.Errorf(INTERNAL_ERROR)
    }

    s.authCfg.haveNewSessions = true
    return nil
}

func (s *Server) RemoveSpotify(w http.ResponseWriter, r *http.Request) error {
    user, err := s.authCfg.database.GetUser(r.Context(), r.Context().Value("username").(string))
    if err != nil && err != sql.ErrNoRows {
       return err
    }

    sessions, err := s.authCfg.database.GetUserMusicSessions(r.Context(), user.ID)
    if err != nil && err != sql.ErrNoRows {
       return err
    }

    var idToRemove int64

    for _, v := range sessions {
        if strings.EqualFold(v.Type, "spotify") {
            if v.Active == 0 {
                return nil
            }
            idToRemove = v.ID
        }
    }

    if idToRemove == 0 {
        return nil
    }

    err = s.authCfg.database.DeactivateMusicSession(r.Context(), idToRemove)
    if err != nil {
        s.log.Error("Removing Music Session", "id", idToRemove, "error", err)
        return fmt.Errorf(INTERNAL_ERROR)
    }

    fmt.Println("Pre REMOVE:", s.authCfg.haveNewSessions)

    s.authCfg.haveNewSessions = true

    fmt.Println("Post REMOVE:", s.authCfg.haveNewSessions)
    return nil
}

func (s *Server) GetLastScrobble(w http.ResponseWriter, r *http.Request) error {
    type Page struct {
        ArtistName string
        TrackName string
        Timestamp string
    }

    username := r.Context().Value("username")

    user, _ := s.authCfg.database.GetUser(r.Context(), username.(string))
    scrobble, _ := s.authCfg.database.GetLatestTrack(r.Context(), user.ID)
    timestamp := time.Unix(0, 0).Add(time.Duration(scrobble.Timestamp) * time.Millisecond)
    page := &Page{
        ArtistName: scrobble.ArtistName,
        TrackName: scrobble.TrackName,
        Timestamp: timestamp.Format("01/02/2006 - 03:04PM"),
    }

    tmpl := template.Must(template.ParseFiles("templates/fragments/last-scrobble.html"))
    tmpl.Execute(w, page)
    return nil
}

func (s *Server) LogUserIn(w http.ResponseWriter, r *http.Request) error {
    r.ParseForm()

    username := r.FormValue("username")
    password := r.FormValue("password")

    if !s.login(r.Context(), username, password) {
        return fmt.Errorf(AUTH_ERROR)
    }

    s.setTokens(w, r, username)
    http.Redirect(w, r, "/settings", http.StatusSeeOther)
    s.log.Info("Login from FE", "username", username, "password", password)
    return nil
}

func (s *Server) Test(w http.ResponseWriter, r *http.Request) error {
    type TestResp struct {
        SuccessResp
        Value string `json:"value"`
    }

    resp := TestResp{ Value: r.Context().Value("username").(string)}
    resp.Success = true
    encode(w, http.StatusOK, resp)
    return nil
}

func (s *Server) TwitterRedirect(w http.ResponseWriter, r *http.Request) error {
    reqToken, verifier, _ := oauth1.ParseAuthorizationCallback(r)
    creds, err :=  s.authCfg.database.GetTwitterSessionByRequestToken(r.Context(), sql.NullString{ String: reqToken, Valid: true })

    if err != nil {
        s.log.Error("Twitter Redirect Mismatch", "error", err)
        return fmt.Errorf(INTERNAL_ERROR)
    }

    accessToken, accessSecret, _ := s.authCfg.TwitterOAuth.AccessToken(reqToken, creds.TwitterRequestSecret.String, verifier)
    s.authCfg.database.SaveTwitterSession(r.Context(), database.SaveTwitterSessionParams{
        TwitterRequestSecret: creds.TwitterRequestSecret,
        TwitterRequestToken: creds.TwitterRequestToken,
        TwitterOauthToken: sql.NullString{ String: accessToken, Valid: true },
        TwitterOauthSecret: sql.NullString{ String: accessSecret, Valid: true },
        Username: creds.Username,
    })

    token := oauth1.NewToken(accessToken, accessSecret)
    s.log.Info("Twitter Auth Redirect", "token", token)
    http.Redirect(w, r, "/settings", http.StatusSeeOther)
    return nil
}

func (s *Server) SpotifyRedirect(w http.ResponseWriter, r *http.Request) error {
    state := r.URL.Query().Get("state")
    username := DecodeRandomState(state)
    session, _ := s.authCfg.database.GetSpotifySession(r.Context(), username)

    if session.SpotifyAuthState.Valid && strings.EqualFold(state, session.SpotifyAuthState.String) {
        res, err := GetSpotifyTokens(r.Context(), r.URL.Query().Get("code"), SpotifyConfig(s.authCfg.config.Spotify))

        if err != nil {
            s.log.Error("Spotify Auth Failue", "err", err)
            return fmt.Errorf(AUTH_ERROR)
        }

        s.log.Info("Spotify Auth Redirect", "response", res)
        s.authCfg.database.SaveSpotifySession(r.Context(), database.SaveSpotifySessionParams{
            SpotifyAccessToken: sql.NullString{ String: res.AccessToken, Valid: true },
            SpotifyRefreshToken: sql.NullString{ String: res.RefreshToken, Valid: true },
            SpotifyID: sql.NullString{ String: res.Id, Valid: true },
            Username: username,
        })
    }

    http.Redirect(w, r, "/settings", http.StatusSeeOther)
    return nil
}

