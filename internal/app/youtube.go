package app

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type Youtube struct {
    service *youtube.Service
    creds struct {
        Key string
    }
}

func NewYoutube(ctx context.Context, key string) *Youtube {
    service, err := youtube.NewService(ctx, option.WithAPIKey(key))
    if err != nil {
        log.Fatalf("Oops: %s", err)
    }

    return &Youtube{
        service: service,
    }
}

func (y *Youtube) Search(term string) string {
    call := y.service.Search.
        List([]string{"id"}).
        Q(term).
        MaxResults(int64(1))

    res, err := call.Do()
    if err != nil {
        log.Fatalf("Oops: %s", err)
    }

    return fmt.Sprintf("https://youtu.be/%s", res.Items[0].Id.VideoId)
}
