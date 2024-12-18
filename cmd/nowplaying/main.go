package main

import (
	"context"
	_ "embed"
	"log"
	"os/signal"
	"syscall"

	"github.com/cg219/nowplaying/internal/app"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    cfg := app.NewConfig()

    go func() {
        if err := app.Run(*cfg); err != nil {
            log.Fatal(err)
        }

    }()

    <- ctx.Done()
    log.Println("Exiting nowplaying safely")
}
