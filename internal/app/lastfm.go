package app

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"time"

	"github.com/cg219/nowplaying/internal/database"
)

type LastFM struct {
    Username string
    Duration time.Duration
    creds struct {
        Name string
        Key string
    }
    config LastFMConfig
    client *http.Client
    db *database.Queries
}

type LastFMSessionResp struct {
    Session struct {
        Name string `json:"name"`
        Key string `json:"key"`
        Subscribers int `json:"subscribers"`
    } `json:"session"`
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

type LastFMConfig struct{
    Key string
    Secret string
}

type apiParam struct {
    Name string
    Value string
}

type tokenResp struct {
    Token string `json:"token"`
}

func NewLastFM(u string, c LastFMConfig, db *database.Queries) *LastFM {
    return &LastFM{
        client: &http.Client{
            Timeout: time.Second * 10,
        },
        Username: u,
        Duration: time.Second * 15,
        db: db,
        config: c,
    }
}

func (l *LastFM) Listen(ctx context.Context) error {
    done := make(chan bool)
    timer := time.NewTicker(l.Duration)
    defer close(done)

    for {
        select {
        case <- done:
            return nil
        case <- timer.C:
            err := l.CheckCurrentTrack(ctx)

            if err != nil {
                log.Printf("Oops: %s\n", err)
                return err
            }
        }
    }
}

func (l *LastFM) Auth(ctx context.Context) error {
    authurl := "http://www.last.fm/api/auth/?api_key=%s&token=%s"
    respBody := tokenResp{}
    req, err := http.NewRequestWithContext(ctx, "GET", l.makeApiUrl("auth.gettoken", nil), nil)
    if err != nil {
        return err
    }

    resp, err := l.client.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    err = json.NewDecoder(resp.Body).Decode(&respBody)
    if err != nil {
        return err
    }

    log.Print("Token: ", respBody.Token)
    exec.Command("open", fmt.Sprintf(authurl, l.config.Key, respBody.Token)).Run()
    fmt.Println("Hit Enter to Continue after authorization")
    fmt.Scanln()

    req, err = http.NewRequestWithContext(ctx, "GET", l.makeApiUrl("auth.getsession", []apiParam{{ Name: "token", Value: respBody.Token }}), nil)
    if err != nil {
        return err
    }

    resp2, err := l.client.Do(req)
    if err != nil {
        return err
    }

    defer resp2.Body.Close()

    var session LastFMSessionResp
    err = json.NewDecoder(resp2.Body).Decode(&session)
    if err != nil {
        return err
    }

    l.db.SaveUser(ctx, l.Username)
    l.db.SaveLastFMSession(ctx, database.SaveLastFMSessionParams{
        LastfmSessionName: sql.NullString{ String: session.Session.Name, Valid: true },
        LastfmSessionKey: sql.NullString{ String: session.Session.Key, Valid: true },
        Username: l.Username,
    })

    l.creds.Name = session.Session.Name
    l.creds.Key = session.Session.Key

    log.Print(session.Session)
    return nil
}

func (l *LastFM) AuthWithDB(ctx context.Context) error {
    dbSession, err := l.db.GetLastFMSession(ctx, l.Username)
    if err != nil {
        log.Printf("Oops: %s\n", err)
    }

    if !dbSession.LastfmSessionKey.Valid && !dbSession.LastfmSessionName.Valid {
        l.Auth(ctx)
        return nil
    }

    l.creds.Name = dbSession.LastfmSessionName.String
    l.creds.Key = dbSession.LastfmSessionKey.String

    return nil
}

func (l *LastFM) CheckCurrentTrack(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", l.makeApiUrl("user.getrecenttracks", []apiParam{
        { Name: "sk", Value: l.creds.Key },
        { Name: "limit", Value: "1" },
        { Name: "user", Value: l.creds.Name },
    }), nil)
    if err != nil {
        return err
    }

    resp, err := l.client.Do(req)
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

func (l *LastFM) makeApiUrl(method string, list []apiParam) string {
    baseurl := "http://ws.audioscrobbler.com/2.0/?format=json&api_sig=%s%s"
    params := ""
    list = append(list, apiParam{ Name: "api_key", Value: l.config.Key })
    list = append(list, apiParam{ Name: "method", Value: method })

    for _, p := range(list) {
        params = fmt.Sprintf("%s&%s=%s", params, p.Name, p.Value)
    }

    log.Printf(baseurl, l.makeSignature(list), params)
    return fmt.Sprintf(baseurl, l.makeSignature(list), params)
}

func (l *LastFM) makeSignature(list []apiParam) string {
    rawSig := ""
    sort.Slice(list, func(i, j int) bool {
        return list[i].Name < list[j].Name
    })

    for _, p := range(list) {
        rawSig = fmt.Sprintf("%s%s%s", rawSig, p.Name, p.Value)
    }

    rawSig = fmt.Sprintf("%s%s", rawSig, l.config.Secret)
    h := md5.New()
    fmt.Fprint(h, rawSig)
    sig := h.Sum(nil)
    // log.Print("Sig: ", rawSig)
    return fmt.Sprintf("%x", sig[:])
}

