package app

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cg219/nowplaying/internal/database"
	"github.com/dghubble/oauth1"
	"github.com/dghubble/oauth1/twitter"
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
        Id int `yaml:"id"`
    } `yaml:"app"`
    Twitter struct {
        Id string `yaml:"id"`
        Secret string `yaml:"secret"`
        Redirect string `yaml:"redirect"`
    }
}

type AppCfg struct {
    config Config
    username string
    ctx context.Context
    LastFMSession *LastFM
    TwitterOAuth oauth1.Config
    listenInterval time.Ticker
    database *database.Queries
    haveNewSessions bool
}

type Session interface {
    AuthWithDB(context.Context) error
    Listen(context.Context, *chan any, chan bool)
    Encode() []byte
    Decode([]byte) error
}

func NewConfig() *Config {
    cfg := &Config{}

    cfg.LastFM.Key = os.Getenv("LASTFM_KEY")
    cfg.LastFM.Secret = os.Getenv("LASTFM_SECRET")
    cfg.Turso.Name = os.Getenv("TURSO_NAME")
    cfg.Turso.Url = os.Getenv("TURSO_URL")
    cfg.Turso.Token = os.Getenv("TURSO_TOKEN")
    cfg.Spotify.Id = os.Getenv("SPOTIFY_ID")
    cfg.Spotify.Secret = os.Getenv("SPOTIFY_SECRET")
    cfg.Spotify.Redirect = os.Getenv("SPOTIFY_REDIRECT")
    cfg.App.Name = os.Getenv("APP_USERNAME")
    cfg.Twitter.Id = os.Getenv("TWITTER_ID")
    cfg.Twitter.Secret = os.Getenv("TWITTER_SECRET")
    cfg.Twitter.Redirect = os.Getenv("TWITTER_REDIRECT")
    cfg.App.Id, _ = strconv.Atoi(os.Getenv("APP_UID"))

    return cfg
}

func AppLoop(cfg *AppCfg) bool {
    cfg.haveNewSessions = false
    ctx := cfg.ctx
    encodedSessions, _ := cfg.database.GetActiveMusicSessions(ctx)
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

    yts := NewYoutube(ctx)
    exit := make(chan struct{})
    output := make(chan any)
    listening := make(chan bool)
    defer close(output)
    defer close(exit)
    defer close(listening)

    for _, s := range sessions {
        go func() {
            s.AuthWithDB(ctx)
            s.Listen(ctx, &output, listening);

            for {
                select {
                case v := <- listening:
                    if !v {
                        log.Println("Something Goin on")
                    }
                    return
                }
            }
        }()
    }

    go func() {
        for {
            if cfg.haveNewSessions {
                exit <- struct{}{}
                return
            }
        }
    }()

    go func() {
        for v := range output {
            switch v := v.(type) {
            case SpotifyListenValue:
                user, _ := cfg.database.GetUser(cfg.ctx, v.Username)
                scrobbler := NewScrobbler(v.Username, cfg.database)
                if ok := scrobbler.Scrobble(context.Background(), Scrobble{
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
                }); ok {
                    twitter := NewTwitter(v.Username, TwitterConfig(cfg.config.Twitter), cfg.database)

                    err := twitter.AuthWithDB(cfg.ctx)
                    if err != nil {
                        log.Printf("Oops: %s", err)
                        continue
                    }

                    playing := fmt.Sprintf("%s - %s", v.Song.Artist, v.Song.Name)
                    tweet := fmt.Sprintf("Now Playing\n\n%s\nLink: %s\n", playing, yts.Search(playing))
                    log.Println(tweet)
                    twitter.Tweet(tweet)
                }

            }
        }
    }()

    <- exit
    
    if cfg.haveNewSessions {
        restartLoop = true
    }

    return restartLoop
}

func Run(config Config) error {
    cfg := &AppCfg{
        config: config,
        username: config.App.Name,
        listenInterval: *time.NewTicker(5 * time.Second),
        ctx: context.Background(),
        TwitterOAuth: oauth1.Config {
            ConsumerKey: config.Twitter.Id,
            ConsumerSecret: config.Twitter.Secret,
            CallbackURL: config.Twitter.Redirect,
            Endpoint: twitter.AuthorizeEndpoint,
        },
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
    provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS("./sql/migrations"))

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

    for {
        restart := AppLoop(cfg)
        if !restart {
            return nil
        }
    }
}
