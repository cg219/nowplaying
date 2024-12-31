package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
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
    cwd, _ := os.Getwd();
    s3cfg, err := config.LoadDefaultConfig(context.Background(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(os.Getenv("R2_KEY"), os.Getenv("R2_SECRET"), "")), config.WithRegion("auto"))

    if err != nil {
        log.Fatal(err)
    }

    client := s3.NewFromConfig(s3cfg, func(o *s3.Options) {
        o.BaseEndpoint = aws.String(os.Getenv("R2_URL"))
    })

    res, err := client.GetObject(context.Background(), &s3.GetObjectInput{
        Bucket: aws.String("nowplaying"),
        Key: aws.String("database.db"),
    })

    if err != nil {
        log.Fatal(err)
    }

    dbfile, err := os.Create(filepath.Join(cwd, os.Getenv("APP_DATA")))
    if err != nil {
        log.Fatal(err)
    }

    data, err := io.ReadAll(res.Body)
    if err != nil {
        res.Body.Close()
        log.Fatal(err)
    }

    res.Body.Close()

    _, err = dbfile.Write(data)
    if err != nil {
        dbfile.Close()
        log.Fatal(err)
    }

    dbfile.Close()

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
        dbfile, err := os.Open(filepath.Join(cwd, os.Getenv("APP_DATA")))

        if err != nil {
            log.Fatal(err)
        }

        var latest bytes.Buffer

        tee := io.TeeReader(dbfile, &latest)
        dbbuf := new(bytes.Buffer)
        io.Copy(dbbuf, tee)
        timestamped := bytes.NewReader(dbbuf.Bytes())
        bckup := bytes.NewReader(latest.Bytes())

        timestamp := time.Now()
        log.Println("Saving database to R2")

        _, err = client.PutObject(context.Background(), &s3.PutObjectInput{
            Bucket: aws.String("nowplaying"),
            Key: aws.String(fmt.Sprintf("%d-database.db", timestamp.UnixMilli())),
            Body: timestamped,
        })

        if err != nil {
            log.Fatal(err.Error())
        }

        _, err = client.PutObject(context.Background(), &s3.PutObjectInput{
            Bucket: aws.String("nowplaying"),
            Key: aws.String("database.db"),
            Body: bckup,
        })

        if err != nil {
            log.Fatal(err.Error())
        }
    }

    log.Println("Exiting nowplaying safely")
}
