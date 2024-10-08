package app

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/cg219/nowplaying/internal/database"
)

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
    encode[TestResp](w, http.StatusOK, resp)
    return nil
}

func (s *Server) SpotifyRedirect(w http.ResponseWriter, r *http.Request) error {
    s.authCfg.SpotifySession.SetAuthCode(r.URL.Query().Get("code"))
    s.authCfg.SpotifySession.GetSpotifyTokens(r.Context())
    fmt.Println(r.URL.Query())
    return nil
}

