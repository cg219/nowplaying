package app

import (
	// "bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strings"
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

type LastFMScrobble struct {
    Artist string `json:"artist"`
    Track string `json:"track"`
    Timestamp string `json:"timestamp"`
    Album string `json:"album,omitempty"`
    TrackNumber string `json:"tracknumber,omitempty"`
    Mbid string `json:"mbid,omitempty"`
    AlbumArtist string `json:"albumArtist,omitempty"`
    Duration string `json:"duration,omitempty"`
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

func (l *LastFM) Listen(ctx context.Context, out *chan any) error {
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
    req := l.makeApiRequest("GET", "auth.gettoken", nil) 

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

    req = l.makeApiRequest("GET", "auth.getsession", []apiParam{{ Name: "token", Value: respBody.Token }})

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

    // l.db.SaveUser(ctx, l.Username)
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

func (l *LastFM) Scrobble(ctx context.Context, sc LastFMScrobble) error {
    params := []apiParam{}
    params = append(params, apiParam{ Name: "artist", Value: sc.Artist })
    params = append(params, apiParam{ Name: "track", Value: sc.Track })
    params = append(params, apiParam{ Name: "timestamp", Value: sc.Timestamp })
    params = append(params, apiParam{ Name: "album", Value: sc.Album })
    params = append(params, apiParam{ Name: "trackNumber", Value: sc.TrackNumber })
    params = append(params, apiParam{ Name: "mbid", Value: sc.Mbid })
    params = append(params, apiParam{ Name: "albumArtist", Value: sc.AlbumArtist })
    params = append(params, apiParam{ Name: "duration", Value: sc.Duration })
    params = append(params, apiParam{ Name: "sk", Value: l.creds.Key })
    req := l.makeApiRequest("POST", "track.scrobble", params)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := l.client.Do(req)
    if err != nil {
        return err
    }
    
    defer resp.Body.Close()

    d, _ := io.ReadAll(resp.Body)

    log.Printf("SE: %s", d)

    // var tracklist map[string]any
    // err = json.NewDecoder(resp.Body).Decode(&tracklist)
    // if err != nil {
    //     log.Println("decode")
    //     return err
    // }
    //
    // log.Println("Output---")
    // for k, v := range tracklist {
    //     log.Printf("%s: %s\n", k, v)
    // }
    // log.Println("---------")

    return nil
}

func (l *LastFM) CheckCurrentTrack(ctx context.Context) error {
    req := l.makeApiRequest("GET", "user.getrecenttracks", []apiParam{
        { Name: "sk", Value: l.creds.Key },
        { Name: "limit", Value: "1" },
        { Name: "user", Value: l.creds.Name },
    })

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

func (l *LastFM) makeApiRequest(action string, method string, list []apiParam) *http.Request {
    params := ""

    if list == nil {
        list = []apiParam{}
    }

    list = append(list, apiParam{ Name: "api_key", Value: l.config.Key })
    list = append(list, apiParam{ Name: "method", Value: method })
    
    body := &url.Values{}

    for _, p := range(list) {
        params = fmt.Sprintf("%s&%s=%s", params, p.Name, url.QueryEscape(p.Value))
        body.Set(p.Name, p.Value)
    }


    if action == "GET" {
        baseurl := "http://ws.audioscrobbler.com/2.0/?format=json&api_sig=%s%s"
        signedUrl := fmt.Sprintf(baseurl, l.makeSignature(list), params)
        log.Printf("Signed URL: %s", signedUrl)
        req, err := http.NewRequest(action, signedUrl, nil)

        if err != nil {
            log.Fatal("Error making LastFM API request")
        }
        
        return req
    }

    baseurl := "http://ws.audioscrobbler.com/2.0/?api_sig=%s"
    signedUrl := fmt.Sprintf(baseurl, l.makeSignature(list))
    req, err := http.NewRequest(action, signedUrl, strings.NewReader(body.Encode()))

    if err != nil {
        log.Fatal("Error making LastFM API request")
    }

    return req
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

