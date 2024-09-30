package app

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/cg219/nowplaying/internal/database"
)

type Spotify struct {
    Username string
    Duration time.Duration
    creds struct {
        AccessToken string
        RefreshToken string
        AuthCode string
    }
    config SpotifyConfig
    client *http.Client
    db *database.Queries
    retrying bool
    youtube *Youtube
}

type SpotifyConfig struct {
    Id string
    Secret string
    Redirect string
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
    Error struct {
        Status json.Number `json:"status"`
        Message string `json:"message"`
    } `json:"error"`
}

func NewSpotify(u string, c SpotifyConfig, db *database.Queries) *Spotify {
    return &Spotify{
        client: &http.Client{
            Timeout: time.Second * 10,
        },
        Username: u,
        Duration: time.Second * 5,
        db: db,
        config: c,
        retrying: false,
        youtube: NewYoutube(context.Background()),
    }
}

func (s *Spotify) SetAuthCode(code string) {
    s.creds.AuthCode = code
}

func (s *Spotify) Listen(ctx context.Context) error {
    done := make(chan bool)
    timer := time.NewTicker(s.Duration)
    defer close(done)

    for {
        select {
        case <- done:
            return nil
        case <- timer.C:
            err := s.CheckCurrentTrack(ctx)

            if err != nil {
                log.Printf("Oops: %s\n", err)
                return err
            }
        }
    }
}

func (s *Spotify) Auth(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", "https://accounts.spotify.com/authorize", nil)
    if err != nil {
        return err
    }

    vals := req.URL.Query()
    vals.Add("response_type", "code")
    vals.Add("client_id", s.config.Id)
    vals.Add("state", "something")
    vals.Add("redirect_uri", s.config.Redirect)
    vals.Add("scope", "user-read-currently-playing user-read-playback-state")

    req.URL.RawQuery = vals.Encode()

    exec.Command("open", req.URL.String()).Run()

    fmt.Println("Hit Enter to Continue after authorization")
    fmt.Scanln()

    return nil
}


func (s *Spotify) GetSpotifyTokens(ctx context.Context) error {
    vals := url.Values{}
    vals.Set("grant_type", "authorization_code")
    vals.Set("code", s.creds.AuthCode)
    vals.Set("redirect_uri", s.config.Redirect)

    req, err := http.NewRequestWithContext(ctx, "POST", "https://accounts.spotify.com/api/token", strings.NewReader(vals.Encode()))
    if err != nil {
        return err
    }

    req.Header.Add("Authorization", fmt.Sprintf("Basic %s", base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", s.config.Id, s.config.Secret)))))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

    resp, err := s.client.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    var data SpotifyTokenResp
    err = json.NewDecoder(resp.Body).Decode(&data)
    if err != nil {
        return err
    }
    
    s.db.SaveUser(ctx, s.Username)
    s.db.SaveSpotifySession(ctx, database.SaveSpotifySessionParams{
        SpotifyAccessToken: sql.NullString{ String: data.AccessToken, Valid: true },
        SpotifyRefreshToken: sql.NullString{ String: data.RefreshToken, Valid: true },
        Username: s.Username,
    })
    
    return nil
}

func (s *Spotify) RefreshSpotifyTokens(ctx context.Context) error {
    vals := url.Values{}
    vals.Set("grant_type", "refresh_token")
    vals.Set("refresh_token", s.creds.RefreshToken)

    req, err := http.NewRequestWithContext(ctx, "POST", "https://accounts.spotify.com/api/token", strings.NewReader(vals.Encode()))
    if err != nil {
        return err
    }

    req.Header.Add("Authorization", fmt.Sprintf("Basic %s", base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", s.config.Id, s.config.Secret)))))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

    resp, err := s.client.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    var data SpotifyTokenResp
    err = json.NewDecoder(resp.Body).Decode(&data)
    if err != nil {
        return err
    }
    
    s.db.UpdateSpotifyAccessToken(ctx, database.UpdateSpotifyAccessTokenParams{
        SpotifyAccessToken: sql.NullString{ String: data.AccessToken, Valid: true },
        Username: s.Username,
    })

    s.creds.AccessToken = data.AccessToken
    return nil
}

func (s *Spotify) AuthWithDB(ctx context.Context) error {
    dbSession, err := s.db.GetSpotifySession(ctx, s.Username)
    if err != nil {
        log.Printf("Oops: %s\n", err)
    }

    if !dbSession.SpotifyAccessToken.Valid && !dbSession.SpotifyRefreshToken.Valid {
        s.Auth(ctx)
        return nil
    }

    s.creds.AccessToken = dbSession.SpotifyAccessToken.String
    s.creds.RefreshToken = dbSession.SpotifyRefreshToken.String

    return nil
}

func (s *Spotify) CheckCurrentTrack(ctx context.Context) error {
    // vals := url.Values{}
    // vals.Set("markets", "US")

    req, err := http.NewRequestWithContext(ctx, "GET", "https://api.spotify.com/v1/me/player/currently-playing", nil)
    if err != nil {
        return err
    }

    req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.creds.AccessToken))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

    resp, err := s.client.Do(req)
    if err != nil {
        return err
    }

    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK {
        var data SpotifyPlayingResp
        err = json.NewDecoder(resp.Body).Decode(&data)
        if err != nil {
            log.Println(err.Error())
            return err
        }

        s.retrying = false
        log.Printf("Spotify: %s - %s\n", data.Item.Artist[0].Name, data.Item.Song)
        log.Printf("Music URL: %s\n", s.youtube.Search(fmt.Sprintf("%s - %s", data.Item.Artist[0].Name, data.Item.Song)))
    } else {
        var data SpotifyPlayingErrorResp
        err = json.NewDecoder(resp.Body).Decode(&data)
        if err != nil {
            if err != io.EOF {
                return err
            }
        }

        if data.Error.Status == "401" && !s.retrying {
            log.Println("Refreshing Spotify Tokens")
            s.retrying = true
            s.RefreshSpotifyTokens(ctx)
            s.CheckCurrentTrack(ctx)
        }
    }

    return nil
}
