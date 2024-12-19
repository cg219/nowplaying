package main

import (
	_ "embed"
	"log"

	"github.com/cg219/nowplaying/internal/app"
)

func main() {
    cfg := app.NewConfig()
    done := make(chan struct{})

    go func() {
        if err := app.Run(*cfg); err != nil {
            log.Fatal(err)
            close(done)
            return
        }
        log.Println("Exiting app func")

        close(done)
    }()

    <- done

    log.Println("Exiting nowplaying safely")
}
