package app

import (
	"context"
	"fmt"
	"log"

	"github.com/ppalone/ytsearch"
)

type Youtube struct {
    service *ytsearch.Client
    creds struct {
        Key string
    }
}

func NewYoutube(ctx context.Context) *Youtube {
    return &Youtube{
        service: &ytsearch.Client{},
    }
}

func (y *Youtube) Search(term string) string {
    res, err := y.service.Search(term)
    if err != nil {
        log.Fatalf("Oops: %s", err)
    }

    return fmt.Sprintf("https://youtu.be/%s", res.Results[0].VideoID)
}
