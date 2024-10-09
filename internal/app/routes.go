package app

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
)

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

