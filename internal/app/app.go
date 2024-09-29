package app

import (
    "context"
    "crypto/md5"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
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
    Subscribers int `json:"subscribers"`
}

type sessionResp struct {
    Session `json:"session"`
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

type apiParam struct {
    Name string
    Value string
}

type AuthCfg struct {
    key string
    secret string
    username string
    client *http.Client
    ctx context.Context
    session *Session
    listenInterval time.Ticker
    database *database.Queries
}

func (cfg *AuthCfg) Listen() {
    done := make(chan bool)

    for {
        select {
        case <- done:
            return
        case <- cfg.listenInterval.C:
            err := cfg.CheckCurrentTrack()

            if err != nil {
                log.Printf("Oops: %s\n", err)
            }
        }
    }
}

func (cfg *AuthCfg) Auth() error {
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
    exec.Command("open", fmt.Sprintf(authurl, cfg.key, respBody.Token)).Run()
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

    var session sessionResp
    err = json.NewDecoder(resp2.Body).Decode(&session)
    if err != nil {
        return err
    }

    cfg.database.SaveUser(cfg.ctx, cfg.username)
    cfg.database.SaveLastFMSession(cfg.ctx, database.SaveLastFMSessionParams{
        LastfmSessionName: sql.NullString{ String: session.Name, Valid: true },
        LastfmSessionKey: sql.NullString{ String: session.Key, Valid: true },
        Username: cfg.username,
    })

    cfg.session = &session.Session
    log.Print(session.Session)
    return nil
}

func (cfg *AuthCfg) AuthWithDB() error {
    dbSession, err := cfg.database.GetLastFMSession(cfg.ctx, cfg.username)
    if err != nil {
        log.Printf("Oops: %s\n", err)
    }

    if !dbSession.LastfmSessionKey.Valid && !dbSession.LastfmSessionName.Valid {
        cfg.Auth()
        return nil
    }

    cfg.session = &Session{ Name: dbSession.LastfmSessionName.String, Key: dbSession.LastfmSessionKey.String }
    return nil
}

func (cfg *AuthCfg) CheckCurrentTrack() error {
    req, err := http.NewRequestWithContext(cfg.ctx, "GET", cfg.MakeApiUrl("user.getrecenttracks", []apiParam{
        { Name: "sk", Value: cfg.session.Key },
        { Name: "limit", Value: "1" },
        { Name: "user", Value: cfg.session.Name },
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

    log.Print(tracklist.Recent.Tracks[0].Name)
    return nil
}

func (cfg *AuthCfg) MakeApiUrl(method string, list []apiParam) string {
    baseurl := "http://ws.audioscrobbler.com/2.0/?format=json&api_sig=%s%s"
    params := ""
    list = append(list, apiParam{ Name: "api_key", Value: cfg.key })
    list = append(list, apiParam{ Name: "method", Value: method })

    for _, p := range(list) {
        params = fmt.Sprintf("%s&%s=%s", params, p.Name, p.Value)
    }

    log.Printf(baseurl, makeSignature(cfg.secret, list), params)
    return fmt.Sprintf(baseurl, makeSignature(cfg.secret, list), params)
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
        key: config.LastFM.Key, 
        secret: config.LastFM.Secret,
        username: config.App.Name,
        client: &http.Client{
            Timeout: time.Second * 60,
        },
        session: nil,
        listenInterval: *time.NewTicker(15 * time.Second),
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

    if err := cfg.AuthWithDB(); err != nil {
        return err
    }

    cfg.Listen()

    return nil
}
