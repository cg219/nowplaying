package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

    if strings.EqualFold(os.Getenv("APP_EXIT_BACKUP"), "1") {
        s3cfg, err := config.LoadDefaultConfig(context.Background(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(os.Getenv("R2_KEY"), os.Getenv("R2_SECRET"), "")), config.WithRegion("auto"))
        if err != nil {
            log.Fatal(err)
        }

        client := s3.NewFromConfig(s3cfg, func(o *s3.Options) {
            o.BaseEndpoint = aws.String(os.Getenv("R2_URL"))
        })

        cwd, _ := os.Getwd();
        dbfile, err := os.Open(filepath.Join(cwd, os.Getenv("APP_DATA")))
        if err != nil {
            log.Fatal(err)
        }

        timestamp := time.Now()
        res, err := client.PutObject(context.Background(), &s3.PutObjectInput{
            Bucket: aws.String("nowplaying"),
            Key: aws.String(fmt.Sprintf("%d-database.db", timestamp.UnixMilli())),
            Body: dbfile,
        })

        if err != nil {
            log.Fatal(err)
        }

        log.Println(res)
    }

    log.Println("Exiting nowplaying safely")
}
