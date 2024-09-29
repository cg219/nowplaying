package app

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cg219/nowplaying/internal/database"
	"github.com/pressly/goose/v3"
	"github.com/tursodatabase/go-libsql"
)

type Config struct {
    LastFM struct {
        Key string `yaml:"key"`
        Secret string `yaml:"secret"`
    } `yaml:"lastfm"`
    Turso struct {
        Name string `yaml:"name"`
        Url string `yaml:"url"`
        Token string `yaml:"token"`
    }
    Spotify struct {
        Id string `yaml:"id"`
        Secret string `yaml:"secret"`
        Redirect string `yaml:"redirect"`
    } `yaml:"spotify"`
    App struct {
        Name string `yaml:"name"`
    } `yaml:"app"`
}

type tokenResp struct {
    Token string `json:"token"`
}

type Session struct {
    Name string `json:"name"`
    Key string `json:"key"`
    AccessToken string
    RefreshToken string
    Subscribers int `json:"subscribers"`
}

type LastFMSessionResp struct {
    Session Session `json:"session"`
}

type LastFMArtist struct {
    Mbid string `json:"mbid"`
    Name string `json:"#text"`
}

type LastFMImage struct {
    Size string `json:"size"`
    Url string `json:"#text"`
}

type LastFMAlbum struct {
    Mbid string `json:"mbid"`
    Name string `json:"#text"`
}

type LastFMDate struct {
    Seconds json.Number `json:"uts"`
    DateString string `json:"#text"`
}

type LastFMTrack struct {
    Artist LastFMArtist `json:"artist"`
    Images []LastFMImage `json:"image"`
    Album LastFMAlbum `json:"album"`
    Date LastFMDate `json:"date"`
    Name string `json:"name"`
    Url string `json:"url"`
}

type LastFMCurrentTrackResp struct {
    Recent struct {
        Tracks []LastFMTrack `json:"track"`
    } `json:"recenttracks"`
}

type SpotifyTokenResp struct {
    AccessToken string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    Scope string `json:"scope"`
}

type SpotifyPlayingResp struct {
    Timestamp json.Number `json:"timestamp"`
    Progress json.Number `json:"progress_ms"`
    Item struct {
        Album struct {
            Name string `json:"name"`
        } `json:"album"`
        Artist []struct{ Name string `json:"name"`} `json:"artists"`
        Song string `json:"name"`
    } `json:"item"`
}

type SpotifyPlayingErrorResp struct {
    Status json.Number `json:"status"`
    Message string `json:"message"`
}

type apiParam struct {
    Name string
    Value string
}

type AuthCfg struct {
    config Config
    username string
    client *http.Client
    ctx context.Context
    lastfmSession *Session
    spotifySession *Session
    listenInterval time.Ticker
    database *database.Queries
}

func (cfg *AuthCfg) ListenToLastFM() {
    done := make(chan bool)

    for {
        select {
        case <- done:
            return
        case <- cfg.listenInterval.C:
            err := cfg.CheckCurrentLastFMTrack()

            if err != nil {
                log.Printf("Oops: %s\n", err)
            }
        }
    }
}

func (cfg *AuthCfg) ListenToSpotify() {
    done := make(chan bool)

    for {
        select {
        case <- done:
            return
        case <- cfg.listenInterval.C:
            err := cfg.CheckCurrentSpotifyTrack()

            if err != nil {
                log.Printf("Oops: %s\n", err)
            }
        }
    }
}

func (cfg *AuthCfg) Listen() {
    done := make(chan bool)

    for {
        select {
        case <- done:
            return
        case <- cfg.listenInterval.C:
            err := cfg.CheckCurrentSpotifyTrack()

            if err != nil {
                log.Printf("Oops: %s\n", err)
            }

            err = cfg.CheckCurrentLastFMTrack()

            if err != nil {
                log.Printf("Oops: %s\n", err)
            }
        }
    }
}

func (cfg *AuthCfg) AuthLastFM() error {
    authurl := "http://www.last.fm/api/auth/?api_key=%s&token=%s"
    respBody := tokenResp{}
    req, err := http.NewRequestWithContext(cfg.ctx, "GET", cfg.MakeApiUrl("auth.gettoken", nil), nil)
    if err != nil {
        return err
    }

    resp, err := cfg.client.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    err = json.NewDecoder(resp.Body).Decode(&respBody)
    if err != nil {
        return err
    }

    log.Print("Token: ", respBody.Token)
    exec.Command("open", fmt.Sprintf(authurl, cfg.config.LastFM.Key, respBody.Token)).Run()
    fmt.Println("Hit Enter to Continue after authorization")
    fmt.Scanln()

    req, err = http.NewRequestWithContext(cfg.ctx, "GET", cfg.MakeApiUrl("auth.getsession", []apiParam{{ Name: "token", Value: respBody.Token }}), nil)
    if err != nil {
        return err
    }

    resp2, err := cfg.client.Do(req)
    if err != nil {
        return err
    }

    defer resp2.Body.Close()

    var session LastFMSessionResp
    err = json.NewDecoder(resp2.Body).Decode(&session)
    if err != nil {
        return err
    }

    cfg.database.SaveUser(cfg.ctx, cfg.username)
    cfg.database.SaveLastFMSession(cfg.ctx, database.SaveLastFMSessionParams{
        LastfmSessionName: sql.NullString{ String: session.Session.Name, Valid: true },
        LastfmSessionKey: sql.NullString{ String: session.Session.Key, Valid: true },
        Username: cfg.username,
    })

    cfg.lastfmSession = &session.Session
    log.Print(session.Session)
    return nil
}

func (cfg *AuthCfg) AuthSpotify() error {
    req, err := http.NewRequestWithContext(cfg.ctx, "GET", "https://accounts.spotify.com/authorize", nil)
    if err != nil {
        return err
    }

    vals := req.URL.Query()
    vals.Add("response_type", "code")
    vals.Add("client_id", cfg.config.Spotify.Id)
    vals.Add("state", "something")
    vals.Add("redirect_uri", cfg.config.Spotify.Redirect)
    vals.Add("scope", "user-read-currently-playing user-read-playback-state")

    req.URL.RawQuery = vals.Encode()

    exec.Command("open", req.URL.String()).Run()

    fmt.Println("Hit Enter to Continue after authorization")
    fmt.Scanln()

    return nil
}


func (cfg *AuthCfg) GetSpotifyTokens(code string) error {
    vals := url.Values{}
    vals.Set("grant_type", "authorization_code")
    vals.Set("code", code)
    vals.Set("redirect_uri", cfg.config.Spotify.Redirect)

    req, err := http.NewRequestWithContext(cfg.ctx, "POST", "https://accounts.spotify.com/api/token", strings.NewReader(vals.Encode()))
    if err != nil {
        return err
    }

    req.Header.Add("Authorization", fmt.Sprintf("Basic %s", base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.config.Spotify.Id, cfg.config.Spotify.Secret)))))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

    resp, err := cfg.client.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    var data SpotifyTokenResp
    err = json.NewDecoder(resp.Body).Decode(&data)
    if err != nil {
        return err
    }
    
    cfg.database.SaveUser(cfg.ctx, cfg.username)
    cfg.database.SaveSpotifySession(cfg.ctx, database.SaveSpotifySessionParams{
        SpotifyAccessToken: sql.NullString{ String: data.AccessToken, Valid: true },
        SpotifyRefreshToken: sql.NullString{ String: data.RefreshToken, Valid: true },
        Username: cfg.username,
    })
    
    return nil
}

func (cfg *AuthCfg) RefreshSpotifyTokens() error {
    vals := url.Values{}
    vals.Set("grant_type", "refresh_token")
    vals.Set("refresh_token", cfg.spotifySession.RefreshToken)

    req, err := http.NewRequestWithContext(cfg.ctx, "POST", "https://accounts.spotify.com/api/token", strings.NewReader(vals.Encode()))
    if err != nil {
        return err
    }

    req.Header.Add("Authorization", fmt.Sprintf("Basic %s", base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.config.Spotify.Id, cfg.config.Spotify.Secret)))))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

    resp, err := cfg.client.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    var data SpotifyTokenResp
    err = json.NewDecoder(resp.Body).Decode(&data)
    if err != nil {
        return err
    }
    
    cfg.database.SaveSpotifySession(cfg.ctx, database.SaveSpotifySessionParams{
        SpotifyAccessToken: sql.NullString{ String: data.AccessToken, Valid: true },
        SpotifyRefreshToken: sql.NullString{ String: data.RefreshToken, Valid: true },
        Username: cfg.username,
    })

    cfg.spotifySession = &Session{ AccessToken: data.AccessToken, RefreshToken: data.RefreshToken }

    return nil
}

func (cfg *AuthCfg) AuthLastFMWithDB() error {
    dbSession, err := cfg.database.GetLastFMSession(cfg.ctx, cfg.username)
    if err != nil {
        log.Printf("Oops: %s\n", err)
    }

    if !dbSession.LastfmSessionKey.Valid && !dbSession.LastfmSessionName.Valid {
        cfg.AuthLastFM()
        return nil
    }

    cfg.lastfmSession = &Session{ Name: dbSession.LastfmSessionName.String, Key: dbSession.LastfmSessionKey.String }
    return nil
}

func (cfg *AuthCfg) AuthSpotifyWithDB() error {
    dbSession, err := cfg.database.GetSpotifySession(cfg.ctx, cfg.username)
    if err != nil {
        log.Printf("Oops: %s\n", err)
    }

    if !dbSession.SpotifyAccessToken.Valid && !dbSession.SpotifyRefreshToken.Valid {
        cfg.AuthSpotify()
        return nil
    }

    cfg.spotifySession = &Session{ AccessToken: dbSession.SpotifyAccessToken.String, RefreshToken: dbSession.SpotifyRefreshToken.String }
    return nil
}

func (cfg *AuthCfg) CheckCurrentLastFMTrack() error {
    req, err := http.NewRequestWithContext(cfg.ctx, "GET", cfg.MakeApiUrl("user.getrecenttracks", []apiParam{
        { Name: "sk", Value: cfg.lastfmSession.Key },
        { Name: "limit", Value: "1" },
        { Name: "user", Value: cfg.lastfmSession.Name },
    }), nil)
    if err != nil {
        return err
    }

    resp, err := cfg.client.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    var tracklist LastFMCurrentTrackResp 
    err = json.NewDecoder(resp.Body).Decode(&tracklist)
    if err != nil {
        return err
    }

    log.Printf("LastFM: %s - %s\n",tracklist.Recent.Tracks[0].Artist.Name, tracklist.Recent.Tracks[0].Name)
    return nil
}

func (cfg *AuthCfg) CheckCurrentSpotifyTrack() error {
    // vals := url.Values{}
    // vals.Set("markets", "US")

    req, err := http.NewRequestWithContext(cfg.ctx, "GET", "https://api.spotify.com/v1/me/player/currently-playing", nil)
    if err != nil {
        return err
    }

    req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cfg.spotifySession.AccessToken))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

    resp, err := cfg.client.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK {
        var data SpotifyPlayingResp
        err = json.NewDecoder(resp.Body).Decode(&data)
        if err != nil {
            return err
        }

        log.Printf("Spotify: %s - %s\n", data.Item.Artist[0].Name, data.Item.Song)
    } else {
        var data SpotifyPlayingErrorResp
        err = json.NewDecoder(resp.Body).Decode(&data)
        if err != nil {
            if err != io.EOF {
                return err
            }
        }

        if data.Status == json.Number(401) {
            log.Println("Refreshing Spotify Tokens")
            cfg.RefreshSpotifyTokens()
        }
    }

    return nil
}

func (cfg *AuthCfg) MakeApiUrl(method string, list []apiParam) string {
    baseurl := "http://ws.audioscrobbler.com/2.0/?format=json&api_sig=%s%s"
    params := ""
    list = append(list, apiParam{ Name: "api_key", Value: cfg.config.LastFM.Key })
    list = append(list, apiParam{ Name: "method", Value: method })

    for _, p := range(list) {
        params = fmt.Sprintf("%s&%s=%s", params, p.Name, p.Value)
    }

    log.Printf(baseurl, makeSignature(cfg.config.LastFM.Secret, list), params)
    return fmt.Sprintf(baseurl, makeSignature(cfg.config.LastFM.Secret, list), params)
}

func makeSignature (secret string, list []apiParam) string {
    rawSig := ""
    sort.Slice(list, func(i, j int) bool {
        return list[i].Name < list[j].Name
    })

    for _, p := range(list) {
        rawSig = fmt.Sprintf("%s%s%s", rawSig, p.Name, p.Value)
    }

    rawSig = fmt.Sprintf("%s%s", rawSig, secret)
    h := md5.New()
    fmt.Fprint(h, rawSig)
    sig := h.Sum(nil)
    // log.Print("Sig: ", rawSig)
    return fmt.Sprintf("%x", sig[:])
}

// TODO:
// - Lookup track on youtube for a link
// - Setup ai to verify the youtube link
func Run(config Config) error {
    cfg := &AuthCfg{
        config: config,
        username: config.App.Name,
        client: &http.Client{
            Timeout: time.Second * 60,
        },
        spotifySession: nil,
        lastfmSession: nil,
        listenInterval: *time.NewTicker(5 * time.Second),
        ctx: context.Background(),
    }

    dbName := config.Turso.Name
    dbUrl := config.Turso.Url
    dbAuthToken := config.Turso.Token
    tmp, err := os.MkdirTemp("", "libdata-*")

    if  err != nil {
        return err
    }

    defer os.RemoveAll(tmp)
    dbPath := filepath.Join(tmp, dbName)
    conn, err := libsql.NewEmbeddedReplicaConnector(dbPath, dbUrl, libsql.WithAuthToken(dbAuthToken))

    if err != nil {
        return err
    }

    defer conn.Close()
    db := sql.OpenDB(conn)
    defer db.Close()
    provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS("./migrations"))

    if err != nil {
        return err
    }

    results, err := provider.Up(context.Background())

    if err != nil {
        return err
    }

    for _, r := range results {
        log.Println("goose: %s, %s", r.Source.Path, r.Duration)
    }

    cfg.database = database.New(db)

    go func() {
        StartServer(cfg)
    }()

    //TODO: Properly wait for server to start befor running auth. Temp fix to start server first when testing locally
    time.Sleep(time.Second * 4)

    if err := cfg.AuthLastFMWithDB(); err != nil {
        return err
    }

    if err := cfg.AuthSpotifyWithDB(); err != nil {
        return err
    }

    cfg.Listen()

    return nil
}
