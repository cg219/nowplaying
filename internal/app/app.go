package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/cg219/nowplaying/internal/database"
	"github.com/dghubble/oauth1"
	"github.com/dghubble/oauth1/twitter"
	"github.com/pressly/goose/v3"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
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
    Data struct {
        Path string `yaml:"data"`
    } `yaml:"app"`
    Spotify struct {
        Id string `yaml:"id"`
        Secret string `yaml:"secret"`
        Redirect string `yaml:"redirect"`
    } `yaml:"spotify"`
    Twitter struct {
        Id string `yaml:"id"`
        Secret string `yaml:"secret"`
        Redirect string `yaml:"redirect"`
    } `yaml:"twitter"`
    Discogs struct {
        Key string `yaml:"key"`
        Secret string `yaml:"secret"`
    } `json:"discogs"`
    R2 struct {
        Key string `yaml:"key"`
        Secret string `yaml:"secret"`
        Token string `yaml:"token"`
        Url string `yaml:"url"`
    } `yaml:"r2"`
    Frontend embed.FS
    Migrations embed.FS
}

type AppCfg struct {
    config Config
    LastFMSession *LastFM
    TwitterOAuth oauth1.Config
    listenInterval time.Ticker
    database *database.Queries
    haveNewSessions bool
    subscribers map[int64]Subscriber
    scrobbles chan ScrobblePack
    subMutex sync.RWMutex
}

type ScrobblePack struct {
    Scrobble Scrobble
    Username string
}

type Session interface {
    AuthWithDB(context.Context) error
    Listen(context.Context, *chan any, chan bool)
    Encode() []byte
    Decode([]byte) error
}

type Publisher interface {
    Register(sub Subscriber) int64
    Unregister(id int64)
    Notify(scrobble Scrobble, username string)
}

type Subscriber interface {
    Execute(scrobble Scrobble, username string)
}

func (cfg *AppCfg) Register(sub Subscriber) int64 {
    id, err := rand.Int(rand.Reader, big.NewInt(100000))
    if err != nil {
        log.Printf("err creating random int %s", err.Error())
        return -1
    }

    cfg.subMutex.Lock()
    defer cfg.subMutex.Unlock()

    cfg.subscribers[id.Int64()] = sub
    return id.Int64()
}

func (cfg *AppCfg) Unregister(id int64) {
    cfg.subMutex.Lock()
    defer cfg.subMutex.Unlock()

    delete(cfg.subscribers, id)
}

func (cfg *AppCfg) Notify(scrobble Scrobble, username string) {
    cfg.subMutex.RLock()
    defer cfg.subMutex.RUnlock()

    for _, sub := range cfg.subscribers {
        sub.Execute(scrobble, username)
    }
}

func NewConfig(frontend embed.FS, migrations embed.FS) *Config {
    cfg := &Config{}

    cfg.LastFM.Key = os.Getenv("LASTFM_KEY")
    cfg.LastFM.Secret = os.Getenv("LASTFM_SECRET")
    cfg.Turso.Name = os.Getenv("TURSO_NAME")
    cfg.Turso.Url = os.Getenv("TURSO_URL")
    cfg.Turso.Token = os.Getenv("TURSO_TOKEN")
    cfg.Spotify.Id = os.Getenv("SPOTIFY_ID")
    cfg.Spotify.Secret = os.Getenv("SPOTIFY_SECRET")
    cfg.Spotify.Redirect = os.Getenv("SPOTIFY_REDIRECT")
    cfg.Twitter.Id = os.Getenv("TWITTER_ID")
    cfg.Twitter.Secret = os.Getenv("TWITTER_SECRET")
    cfg.Twitter.Redirect = os.Getenv("TWITTER_REDIRECT")
    cfg.Discogs.Key = os.Getenv("DISCOGS_KEY")
    cfg.Discogs.Secret = os.Getenv("DISCOGS_SECRET")
    cfg.R2.Key = os.Getenv("R2_KEY")
    cfg.R2.Secret = os.Getenv("R2_SECRET")
    cfg.R2.Token = os.Getenv("R2_TOKEN")
    cfg.R2.Url = os.Getenv("R2_URL")
    cfg.Data.Path = os.Getenv("APP_DATA")
    cfg.Frontend = frontend
    cfg.Migrations = migrations

    return cfg
}

func NewConfigFromSecrets(data []byte, frontend embed.FS, migrations embed.FS) *Config {
    cfg := &Config{}

    if err := yaml.Unmarshal(data, cfg); err != nil {
        log.Fatal("Error unmarshalling secrets file")
    }

    cfg.Frontend = frontend
    cfg.Migrations = migrations

    return cfg
}

func AppLoop(cfg *AppCfg) bool {
    cfg.haveNewSessions = false
    encodedSessions, _ := cfg.database.GetActiveMusicSessions(context.Background())
    sessions := []Session{}
    restartLoop := false

    for _, es := range encodedSessions {
        d, _ := base64.StdEncoding.DecodeString(es.Data)

        switch es.Type {
        case "spotify":
            s := NewSpotifyFromEncoded(d, SpotifyConfig(cfg.config.Spotify), cfg.database)
            s.Id = int(es.ID)
            sessions = append(sessions, s)
        case `applemusic`:
            fmt.Println("Apple Music: ")
        }
    }

    exit := make(chan struct{})
    output := make(chan any)
    listening := make(chan bool)
    defer func() {
        _, ok := <- exit
        if ok {
            close(exit)
        }

        _, ok = <- output
        if ok {
            close(output)
        }

        _, ok = <- listening
        if ok {
            close(listening)
        }
    }()

    for si, s := range sessions {
        go func() {
            s.AuthWithDB(context.Background())
            s.Listen(context.Background(), &output, listening);

            <- listening

            log.Println("Something Goin on")
            sessions = slices.Delete(sessions, si, si + 1)
        }()
    }

    go func() {
        for {
            if cfg.haveNewSessions {
                log.Println("new session added")
                exit <- struct{}{}
                return
            }
        }
    }()

    go func() {
        for v := range output {
            switch v := v.(type) {
            case SpotifyListenValue:
                user, _ := cfg.database.GetUser(context.Background(), v.Username)
                scrobbler := NewScrobbler(v.Username, cfg.database)
                scrobble := Scrobble{
                    ArtistName: v.Song.Artist,
                    TrackName: v.Song.Name,
                    AlbumName: v.Song.Album.Name,
                    AlbumArtist: v.Song.Album.Artist,
                    Timestamp: v.Song.Timestamp,
                    Duration: v.Song.Duration,
                    TrackNumber: fmt.Sprintf("%d", v.Song.TrackNumber),
                    Source: "spotify-local",
                    Uid: int(user.ID),
                    Progress: v.Song.Progress,
                }

                if ok := scrobbler.Scrobble(context.Background(), scrobble); ok {
                    log.Printf("SCROBBLED: %s - %s\n", v.Song.Artist, v.Song.Name)
                    cfg.Notify(scrobble, v.Username)
                }
            }
        }
    }()

    <- exit
    log.Println("Exiting AppLoop()")

    if cfg.haveNewSessions {
        restartLoop = true
    }
    return restartLoop
}

func Run(config Config) error {
    cfg := &AppCfg{
        config: config,
        listenInterval: *time.NewTicker(5 * time.Second),
        TwitterOAuth: oauth1.Config {
            ConsumerKey: config.Twitter.Id,
            ConsumerSecret: config.Twitter.Secret,
            CallbackURL: config.Twitter.Redirect,
            Endpoint: twitter.AuthorizeEndpoint,
        },
        subscribers: make(map[int64]Subscriber), 
        scrobbles: make(chan ScrobblePack, 100),
    }

    cwd, _ := os.Getwd();
    db, err := sql.Open("sqlite", filepath.Join(cwd, config.Data.Path))
    if err != nil {
        return err
    }

    defer db.Close()

    goose.SetBaseFS(config.Migrations)
    goose.SetDialect("sqlite3")

    if err := goose.Up(db, "sql/migrations"); err != nil {
        return err
    }

    cfg.database = database.New(db)

    go func() {
        StartServer(cfg)
    }()

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    done := make(chan struct{})
    run := make(chan struct{})

    // loop := func() {
    //     log.Println("running AppLoop()")
    //     restart := AppLoop(cfg)
    //
    //     if !restart {
    //         close(done)
    //     } else {
    //         run <- struct{}{}
    //     }
    // }
    //
    // go loop()

    for {
        select {
        case pack := <- cfg.scrobbles:
            scrobble := pack.Scrobble
            username := pack.Username
            user, _ := cfg.database.GetUser(context.Background(), username)
            scrobbler := NewScrobbler(username, cfg.database)
            scrobble.Uid = int(user.ID)

            if ok := scrobbler.Scrobble(context.Background(), scrobble); ok {
                log.Printf("SCROBBLED: %s - %s\n", scrobble.ArtistName, scrobble.TrackName)
                cfg.Notify(scrobble, username)
            }
        case <- ctx.Done():
            log.Println("terminating Run()")
            return nil
        case <-done:
            log.Println("AppLoop() complete")
            return nil
        case <-run:
            log.Println("restart AppLoop()")
            // go loop()
        }
    }
}
