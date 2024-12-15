package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/cg219/nowplaying/internal/database"
	"github.com/dghubble/oauth1"
)

func (s *Server) ResetPassword(w http.ResponseWriter, r *http.Request) error {
    resettimer := time.Now().Unix()
    type Body struct {
        Username string `json:"username"`
        Reset string `json:"reset"`
        Password string `json:"password"`
        PasswordConfirm string `json:"passwordConfirm"`
    }

    body, err := decode[Body](r)
    if err != nil {
        return err
    }

    if !strings.EqualFold(body.Password, body.PasswordConfirm) {
        return fmt.Errorf(AUTH_ERROR)
    }

    hashPass, _ := s.hasher.EncodeFromString(body.Password)

    s.authCfg.database.ResetPassword(r.Context(), database.ResetPasswordParams{
        Reset: sql.NullString{ String: body.Reset, Valid: true },
        ResetTime: sql.NullInt64{ Int64: resettimer, Valid: true },
        Password: hashPass,
    })

    data := SuccessResp{ Success: true }

    encode(w, 200, data)
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

    // TODO: Setup email service to send this to user email
    s.log.Info("Reset Link:", "url", fmt.Sprintf("http://localhost:%s/reset/%s", "3006", reset))

    return nil
}

func (s *Server) ShareTopArtists(w http.ResponseWriter, r *http.Request) error {
    username := r.Context().Value("username").(string)
    twitter := NewTwitter(username, TwitterConfig(s.authCfg.config.Twitter), s.authCfg.database)
    err := twitter.AuthWithDB(context.Background())
    if err != nil {
        s.log.Error("Twitter Auth", "err", err)
        return fmt.Errorf(INTERNAL_ERROR)
    }

    results, _ := s.authCfg.database.GetTopArtistsOfWeek(r.Context(), 7)

    var tweet strings.Builder

    tweet.WriteString("Top artists this week:\n\n")
    for _, artist := range results {
        tweet.WriteString(fmt.Sprintf("%s(%d)\n", artist.Artist, artist.Plays))
    }

    log.Println(tweet.String(), len(tweet.String()))
    twitter.Tweet(tweet.String())
    return nil
}

func (s *Server) ShareTopTracks(w http.ResponseWriter, r *http.Request) error {
    username := r.Context().Value("username").(string)
    twitter := NewTwitter(username, TwitterConfig(s.authCfg.config.Twitter), s.authCfg.database)
    err := twitter.AuthWithDB(context.Background())
    if err != nil {
        s.log.Error("Twitter Auth", "err", err)
        return fmt.Errorf(INTERNAL_ERROR)
    }

    results, _ := s.authCfg.database.GetTopTracksOfWeek(r.Context(), 5)

    var tweet strings.Builder

    tweet.WriteString("Top songs this week:\n\n")
    for _, scrobble := range results {
        tweet.WriteString(fmt.Sprintf("%s - %s(%d)\n", scrobble.ArtistName, scrobble.TrackName, scrobble.Plays))
    }

    log.Println(tweet.String(), len(tweet.String()))
    twitter.Tweet(tweet.String())
    return nil
}

func (s *Server) ShareLatestTrack(w http.ResponseWriter, r *http.Request) error {
    username := r.Context().Value("username").(string)
    user, err := s.authCfg.database.GetUser(r.Context(), username)
    if err != nil && err != sql.ErrNoRows {
        return fmt.Errorf(INTERNAL_ERROR)
    }

    scrobble, _ := s.authCfg.database.GetLatestTrack(r.Context(), user.ID)
    twitter := NewTwitter(username, TwitterConfig(s.authCfg.config.Twitter), s.authCfg.database)
    err = twitter.AuthWithDB(context.Background())
    if err != nil {
        s.log.Error("Twitter Auth", "err", err)
        return fmt.Errorf(INTERNAL_ERROR)
    }

    yts := NewYoutube(r.Context())
    playing := fmt.Sprintf("%s - %s", scrobble.ArtistName, scrobble.TrackName)
    tweet := fmt.Sprintf("Now Playing\n\n%s\nLink: %s\n", playing, yts.Search(playing))
    log.Println(tweet)
    // twitter.Tweet(tweet)
    return nil
}

func (s *Server) AddSpotify(w http.ResponseWriter, r *http.Request) error {
    user, err := s.authCfg.database.GetUser(r.Context(), r.Context().Value("username").(string))
    if err != nil && err != sql.ErrNoRows {
        return fmt.Errorf(INTERNAL_ERROR)
    }

    sessions, err := s.authCfg.database.GetUserMusicSessions(r.Context(), user.ID)
    if err != nil && err != sql.ErrNoRows {
        return fmt.Errorf(INTERNAL_ERROR)
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
        return fmt.Errorf(INTERNAL_ERROR)
    }

    sessions, err := s.authCfg.database.GetUserMusicSessions(r.Context(), user.ID)
    if err != nil && err != sql.ErrNoRows {
        return fmt.Errorf(INTERNAL_ERROR)
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
    type Data struct {
        ArtistName string `json:"artistName"`
        TrackName string `json:"trackName"`
        Timestamp string `json:"timestamp"`
    }

    username := r.Context().Value("username")
    user, _ := s.authCfg.database.GetUser(r.Context(), username.(string))
    scrobble, _ := s.authCfg.database.GetLatestTrack(r.Context(), user.ID)
    timestamp := time.Unix(0, 0).Add(time.Duration(scrobble.Timestamp) * time.Millisecond)
    data := Data{
        ArtistName: scrobble.ArtistName,
        TrackName: scrobble.TrackName,
        Timestamp: timestamp.Format("01/02/2006 - 03:04PM"),
    }

    encode(w, 200, data)
    return nil
}

func (s *Server) GetUserData(w http.ResponseWriter, r *http.Request) error {
    type LastScrobble struct {
        ArtistName string `json:"artistName"`
        TrackName string `json:"trackName"`
        Timestamp string `json:"timestamp"`
    }

    type Track struct {
        Name string `json:"name"`
        Track string `json:"track"`
        Plays int `json:"plays"`
    }

    type Data struct {
        LastScrobble LastScrobble `json:"lastScrobble"`
        NavLinks []NavLink `json:"links"`
        Title string `json:"title"`
        Subtitle string `json:"subtitle"`
        Top struct {
            Tracks []Track `json:"tracks"`
            Artists []string `json:"artists"`
        } `json:"top"`
    }

    data := Data{}
    data.Title = "My Page"
    data.Subtitle = "See my activity"
    data.NavLinks = []NavLink{
        { Name: "My Page", Current: true, Url: "/me"},
        { Name: "Settings", Url: "/settings"},
    }

    username := r.Context().Value("username")

    user, _ := s.authCfg.database.GetUser(r.Context(), username.(string))
    scrobble, _ := s.authCfg.database.GetLatestTrack(r.Context(), user.ID)
    tracks, _ := s.authCfg.database.GetTopTracksOfWeek(r.Context(), 10)
    artists, _ := s.authCfg.database.GetTopArtistsOfWeek(r.Context(), 10)
    timestamp := time.Unix(0, 0).Add(time.Duration(scrobble.Timestamp) * time.Millisecond)

    data.Top.Tracks = make([]Track, 0)

    for _, row := range tracks {
        data.Top.Tracks = append(data.Top.Tracks, Track{ Name: row.ArtistName, Plays: int(row.Plays), Track: row.TrackName })
    } 

    for _, row := range artists {
        data.Top.Artists = append(data.Top.Artists, row.Artist)
    } 

    data.LastScrobble = LastScrobble{
        ArtistName: scrobble.ArtistName,
        TrackName: scrobble.TrackName,
        Timestamp: timestamp.Format("01/02/2006 - 03:04PM"),
    }

    encode(w, 200, data)
    return nil
}


func (s *Server) GetSettingsData(w http.ResponseWriter, r *http.Request) error {
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

    type Data struct {
        SpotifyTrack string `json:"spotifyTrack"`
        SpotifyAuthURL string `json:"spotifyUrl"`
        SpotifyOn bool `json:"spotifyOn"`
        TwitterOn bool `json:"twitterOn"`
        TwitterAuthURL string `json:"twitterUrl"`
        NavLinks []NavLink `json:"links"`
        Title string `json:"title"`
        Subtitle string `json:"subtitle"`
    }

    data := Data{}
    data.Title = "Settings"
    data.Subtitle = "Configure your preferences"
    data.NavLinks = []NavLink{
        { Name: "My Page", Url: "/me"},
        { Name: "Settings", Current: true, Url: "/settings"},
    }

    if spotify.SpotifyAccessToken.Valid && spotify.SpotifyRefreshToken.Valid {
        data.SpotifyOn = true
    } else {
        data.SpotifyAuthURL = GetSpotifyAuthURL(r.Context(), user.Username, SpotifyConfig{
            Id: s.authCfg.config.Spotify.Id,
            Redirect: s.authCfg.config.Spotify.Redirect,
            Secret: s.authCfg.config.Spotify.Secret,
        }, s.authCfg.database)
    }

    for _, v := range sessions {
        if strings.EqualFold(v.Type, "spotify") {
            if v.Active == 1 {
                data.SpotifyTrack = "checked"
            }
        }
    }

    if twitter.TwitterOauthToken.Valid && twitter.TwitterOauthSecret.Valid {
        data.TwitterOn = true
    } else {
        data.TwitterAuthURL = GetAuthURL(context.Background(), s.authCfg.TwitterOAuth, s.authCfg.database, user.Username)
    }

    encode(w, 200, data)
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

func (s *Server) LogUserOut(w http.ResponseWriter, r *http.Request) error {
    s.unsetTokens(w, r)
    http.Redirect(w, r, "/", http.StatusSeeOther)
    s.log.Info("Logout from FE")
    return nil
}

func (s *Server) GetResetPasswordData(w http.ResponseWriter, r *http.Request) error {
    type Data struct {
        Valid bool `json:"valid"`
        Username string `json:"username"`
        Reset string `json:"reset"`
    }

    reset := r.PathValue("resetvalue")

    dbValue, _ := s.authCfg.database.CanResetPassword(r.Context(), database.CanResetPasswordParams{
        ResetTime: sql.NullInt64{ Int64: time.Now().Unix(), Valid: true },
        Reset: sql.NullString{ String: reset, Valid: true },
    })

    data := Data{ Valid: dbValue.Valid, Username: dbValue.Username, Reset: reset }

    encode(w, 200, data)
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

