package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
    Id int
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
            Artist []struct{ Name string `json:"name"`} `json:"artists"`
        } `json:"album"`
        Artist []struct{ Name string `json:"name"`} `json:"artists"`
        Song string `json:"name"`
        Duration json.Number `json:"duration_ms"`
        IsLocal bool `json:"is_local"`
        TrackNumber json.Number `json:"track_number"`
        Uri string `json:"uri"`
    } `json:"item"`
    Actions struct {
        Disallows struct {
            Resumed bool `json:"resuming"`
            Paused bool `json:"pausing"`
        } `json:"disallows"`
    } `json:"actions"`
}

type SpotifySong struct {
    Artist string
    Name string
    Album struct {
        Name string
        Artist string
    }
    Progress int
    Duration int
    Timestamp int
    TrackNumber int
}

type SpotifyPlayingErrorResp struct {
    Error struct {
        Status json.Number `json:"status"`
        Message string `json:"message"`
    } `json:"error"`
}

type SpotifyEncoded struct {
    Username string `json:"u"`
    Duration int `json:"d"`
    Retrying bool `json:"r"`
}

type SpotifyListenValue struct {
    Song *SpotifySong
    Username string
}

func NewSpotify(u string, c SpotifyConfig, db *database.Queries) *Spotify {
    return &Spotify{
        client: &http.Client{
            Timeout: time.Second * 10,
        },
        Username: u,
        Duration: time.Second * 4,
        db: db,
        config: c,
        retrying: false,
    }
}

func NewSpotifyFromEncoded(encoded []byte, c SpotifyConfig, db *database.Queries) *Spotify {
    s := &Spotify{
        client: &http.Client{
            Timeout: time.Second * 10,
        },
        db: db,
        config: c,
    }

    s.Decode(encoded)
    return s
}

func NewSpotifySongFromResp(resp SpotifyPlayingResp) *SpotifySong {
    albumArtist := resp.Item.Artist[0].Name

    if len(resp.Item.Album.Artist) > 0 {
        albumArtist = resp.Item.Album.Artist[0].Name
    }

    progress, _ := resp.Progress.Int64()
    duration, _ := resp.Item.Duration.Int64()
    trackNumber, _ := resp.Item.TrackNumber.Int64()
    timestamp, _ := resp.Timestamp.Int64()

    return &SpotifySong{
        Artist: resp.Item.Artist[0].Name,
        Name: resp.Item.Song,
        Album: struct{Name string; Artist string}{
            Name: resp.Item.Album.Name,
            Artist: albumArtist,
        },
        Progress: int(progress),
        Duration: int(duration),
        Timestamp: int(time.Unix(timestamp, 0).Unix()),
        TrackNumber: int(trackNumber),
    }
}

func (s *Spotify) SetAuthCode(code string) {
    s.creds.AuthCode = code
}

func (s *Spotify) Encode() []byte {
    data := &SpotifyEncoded{ Username: s.Username, Retrying: s.retrying,  Duration: int(s.Duration.Milliseconds()) }

    encoded, _ := json.Marshal(data)
    return encoded
}

func (s *Spotify) Decode(encoded []byte) error {
    var data SpotifyEncoded
    err := json.Unmarshal(encoded, &data)
    if err != nil {
        return fmt.Errorf("unmarshal fail: ", err)
    }

    s.Username = data.Username
    s.Duration = time.Duration(data.Duration) * time.Millisecond
    s.retrying = data.Retrying

    return nil
}

func (s *Spotify) Listen(ctx context.Context, out *chan any) error {
    done := make(chan bool)
    timer := time.NewTicker(s.Duration)
    defer close(done)

    for {
        select {
        case <- done:
            return nil
        case <- timer.C:
            song, err := s.CheckCurrentTrack(ctx)

            if err != nil {
                log.Printf("Oops: %s\n", err)
                return err
            }

            if out != nil && song != nil {
                *out <- SpotifyListenValue{ Song: song, Username: s.Username }
            }
        }
    }
}

func GetRandomState(username string) string {
    b := make([]byte, 8)
    rand.Read(b)

   return fmt.Sprintf("%s||%s", username, base64.URLEncoding.EncodeToString(b)[:8])
}

func DecodeRandomState(state string) string {
    return strings.Split(string(state), "||")[0]
}

func GetSpotifyAuthURL(ctx context.Context, username string, config SpotifyConfig, db *database.Queries) string {
    req, _ := http.NewRequestWithContext(ctx, "GET", "https://accounts.spotify.com/authorize", nil)
    state := GetRandomState(username)
    vals := req.URL.Query()

    vals.Add("response_type", "code")
    vals.Add("client_id", config.Id)
    vals.Add("state", state)
    vals.Add("redirect_uri", config.Redirect)
    vals.Add("scope", "user-read-currently-playing user-read-playback-state")
    req.URL.RawQuery = vals.Encode()

    db.SaveSpotifySession(ctx, database.SaveSpotifySessionParams{
        SpotifyAuthState: sql.NullString{ String: state, Valid: true },
        SpotifyAccessToken: sql.NullString{ Valid: false },
        SpotifyRefreshToken: sql.NullString{ Valid: false },
        Username: username,
    })

    return req.URL.String()
}

func (s *Spotify) AuthWithDB(ctx context.Context) error {
    dbSession, err := s.db.GetSpotifySession(ctx, s.Username)

    if (!dbSession.SpotifyAccessToken.Valid && !dbSession.SpotifyRefreshToken.Valid) || err != nil {
        return fmt.Errorf(AUTH_ERROR)
    }

    s.creds.AccessToken = dbSession.SpotifyAccessToken.String
    s.creds.RefreshToken = dbSession.SpotifyRefreshToken.String

    return nil
}

func GetSpotifyTokens(ctx context.Context, code string, config SpotifyConfig) (SpotifyTokenResp, error) {
    var data SpotifyTokenResp

    vals := url.Values{}
    vals.Set("grant_type", "authorization_code")
    vals.Set("code", code)
    vals.Set("redirect_uri", config.Redirect)

    req, _ := http.NewRequestWithContext(ctx, "POST", "https://accounts.spotify.com/api/token", strings.NewReader(vals.Encode()))
    req.Header.Add("Authorization", fmt.Sprintf("Basic %s", base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", config.Id, config.Secret)))))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

    client := &http.Client{
        Timeout: time.Second * 5,
    }
    resp, err := client.Do(req)
    if err != nil {
        return data, err
    }

    defer resp.Body.Close()

    err = json.NewDecoder(resp.Body).Decode(&data)
    if err != nil {
        return data, err
    }
    
    return data, nil
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

func (s *Spotify) CheckCurrentTrack(ctx context.Context) (*SpotifySong, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", "https://api.spotify.com/v1/me/player/currently-playing", nil)
    if err != nil {
        return nil, err
    }

    req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.creds.AccessToken))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

    resp, err := s.client.Do(req)
    if err != nil {
        return nil, err
    }

    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK {
        var data SpotifyPlayingResp
        err = json.NewDecoder(resp.Body).Decode(&data)
        if err != nil {
            log.Println(err.Error())
            return nil, err
        }

        if data.Actions.Disallows.Paused {
            return nil, nil
        }

        song := NewSpotifySongFromResp(data)
        s.retrying = false
        log.Printf("Spotify: %s - %s\n", song.Artist, song.Name)
        return song, nil
    } else {
        var data SpotifyPlayingErrorResp
        err = json.NewDecoder(resp.Body).Decode(&data)
        if err != nil {
            if err != io.EOF {
                return nil, err
            }
        }

        if data.Error.Status == "401" && !s.retrying {
            log.Println("Refreshing Spotify Tokens")
            s.retrying = true
            s.RefreshSpotifyTokens(ctx)
            return s.CheckCurrentTrack(ctx)
        }
    }

    return nil, nil
}

func (s *SpotifySong) String() string {
    return fmt.Sprintf("%s - %s\n%s (by %s)\nTrack: %d", s.Artist, s.Name, s.Album.Name, s.Album.Artist, s.Timestamp)
}
