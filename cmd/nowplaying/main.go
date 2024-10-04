package main

import (
	"context"
	_ "embed"
	"log"
	"os/signal"
	"syscall"

	"github.com/cg219/nowplaying/internal/app"
	"gopkg.in/yaml.v3"
)

//go:embed config.yml
var config string

func main() {
    var cfg app.Config

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    if err := yaml.Unmarshal([]byte(config), &cfg); err != nil {
        log.Fatal(err)
    }

    go func() {
        if err := app.Run(cfg); err != nil {
            log.Fatal(err)
        }

    }()

    <- ctx.Done()
}
