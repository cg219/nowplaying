package app

import (
	"context"
	"database/sql"
	"fmt"

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
}

type TwitterConfig struct {
    Id string
    Secret string
    Redirect string
}

func NewTwitter(c TwitterConfig) *Twitter {
    auth := oauth1.Config {
        ConsumerKey: c.Id,
        ConsumerSecret: c.Secret,
        CallbackURL: c.Redirect,
        Endpoint: twitter.AuthorizeEndpoint,
    }

    twitter := &Twitter{
        config: c,
    }

    twitter.creds.OAuth = auth
    return twitter
}

func (t *Twitter) Auth(ctx context.Context) string {
    reqToken, reqSecret, err := t.creds.OAuth.RequestToken()
    if err != nil {
        fmt.Println(err)
        return ""
    }
    fmt.Println(reqToken)
    authUrl, _ := t.creds.OAuth.AuthorizationURL(reqToken)

    t.creds.RequestSecret = reqSecret

    return authUrl.String()
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
