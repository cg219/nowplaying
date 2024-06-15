package main

import (
	_ "embed"
	"log"

	"github.com/cg219/nowplaying/internal/app"
	"gopkg.in/yaml.v3"
)

//go:embed config.yml
var config string

func main() {
    var cfg app.Config

    if err := yaml.Unmarshal([]byte(config), &cfg); err != nil {
        log.Fatal(err)
    }

    if err := app.Run(cfg); err != nil {
        log.Fatal(err)
    }
}
