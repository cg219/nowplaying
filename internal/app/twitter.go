package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/cg219/nowplaying/internal/database"
	"github.com/dghubble/oauth1"
	"github.com/dghubble/oauth1/twitter"
)

type Twitter struct {
    creds struct {
        AccessToken string
        RequestSecret string
        OAuth oauth1.Config
    }
    config TwitterConfig
    db *database.Queries
    Username string
    client *http.Client
}

type TwitterConfig struct {
    Id string
    Secret string
    Redirect string
}

func NewTwitter(username string, c TwitterConfig, db *database.Queries) *Twitter {
    auth := oauth1.Config {
        ConsumerKey: c.Id,
        ConsumerSecret: c.Secret,
        CallbackURL: c.Redirect,
        Endpoint: twitter.AuthorizeEndpoint,
    }

    twitter := &Twitter{
        config: c,
        Username: username,
        db: db,
    }

    twitter.creds.OAuth = auth
    return twitter
}

func (t *Twitter) Tweet(status string) {
    type post struct {
        Text string `json:"text"`
    }

    body := &post{ Text: status }
    data, _ := json.Marshal(body)

    endpoint := "https://api.x.com/2/tweets"
    req, _ := http.NewRequest("POST", endpoint, bytes.NewReader(data))
    req.Header.Set("Content-Type", "application/json")

    res, err := t.client.Do(req)

    if err != nil {
        log.Fatal(err.Error())
    }

    defer res.Body.Close()

    postRes, _ := io.ReadAll(res.Body)
    fmt.Println("Res:", string(postRes))
}

func (t *Twitter) AuthWithDB(ctx context.Context) error {
    dbSession, err := t.db.GetTwitterSession(ctx, t.Username)

    if (!dbSession.TwitterOauthToken.Valid && !dbSession.TwitterOauthSecret.Valid) || err != nil {
        return fmt.Errorf(AUTH_ERROR)
    }

    config := oauth1.NewConfig(t.config.Id, t.config.Secret)
    token := oauth1.NewToken(dbSession.TwitterOauthToken.String, dbSession.TwitterOauthSecret.String)

    t.client = config.Client(ctx, token)
    return nil
}

func GetAuthURL(ctx context.Context, config oauth1.Config, db *database.Queries, username string) string {
    reqToken, reqSecret, _ := config.RequestToken()
    authUrl, _ := config.AuthorizationURL(reqToken)

    db.SaveTwitterSession(ctx, database.SaveTwitterSessionParams{
        TwitterRequestToken: sql.NullString{ String: reqToken, Valid: true },
        TwitterRequestSecret: sql.NullString{ String: reqSecret, Valid: true },
        Username: username,
    })

    return authUrl.String()
}
