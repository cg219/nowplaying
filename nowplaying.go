package main

import (
	"context"
	"embed"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cg219/nowplaying/internal/app"
)

//go:embed static-app
var Frontend embed.FS

//go:embed sql/migrations/*.sql
var Migrations embed.FS

func main() {
    var cfg *app.Config
    done := make(chan struct{})
    cwd, _ := os.Getwd();

    secretsPath := os.Getenv("NP_CREDTENTIALS")
    _, err := os.Stat(secretsPath)

    if err != nil {
        if os.IsNotExist(err) {
            log.Printf("secrets file not found: %s\nFalling back to env variables\n", secretsPath)
            cfg = app.NewConfig(Frontend, Migrations)
        } else if os.IsPermission(err) {
            log.Printf("incorrect permissions on secret file: %s\nFalling back to env variables\n", secretsPath)
            cfg = app.NewConfig(Frontend, Migrations)
        } else {
            log.Fatal(err)
        }
    } else {
        data, err := os.ReadFile(secretsPath)
        if err != nil {
            log.Printf("error loading secrets file: %s; err: %s\nFalling back to env variables\n", secretsPath, err.Error())
        }

        cfg = app.NewConfigFromSecrets(data, Frontend, Migrations)
    }

    s3cfg, err := config.LoadDefaultConfig(context.Background(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.R2.Key, cfg.R2.Secret, "")), config.WithRegion("auto"))

    if err != nil {
        log.Fatal(err)
    }

    client := s3.NewFromConfig(s3cfg, func(o *s3.Options) {
        o.BaseEndpoint = aws.String(cfg.R2.Url)
    })

    res, err := client.GetObject(context.Background(), &s3.GetObjectInput{
        Bucket: aws.String("nowplaying"),
        Key: aws.String("database.db"),
    })

    if err != nil {
        log.Fatal(err)
    }

    dbfile, err := os.Create(filepath.Join(cwd, cfg.Data.Path))
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

    // if strings.EqualFold(os.Getenv("APP_EXIT_BACKUP"), "1") {
    //     dbfile, err := os.Open(filepath.Join(cwd, cfg.Data.Path))
    //
    //     if err != nil {
    //         log.Fatal(err)
    //     }
    //
    //     var latest bytes.Buffer
    //
    //     tee := io.TeeReader(dbfile, &latest)
    //     dbbuf := new(bytes.Buffer)
    //     io.Copy(dbbuf, tee)
    //     timestamped := bytes.NewReader(dbbuf.Bytes())
    //     bckup := bytes.NewReader(latest.Bytes())
    //
    //     timestamp := time.Now()
    //     log.Println("Saving database to R2")
    //
    //     _, err = client.PutObject(context.Background(), &s3.PutObjectInput{
    //         Bucket: aws.String("nowplaying"),
    //         Key: aws.String(fmt.Sprintf("%d-database.db", timestamp.UnixMilli())),
    //         Body: timestamped,
    //     })
    //
    //     if err != nil {
    //         log.Fatal(err.Error())
    //     }
    //
    //     _, err = client.PutObject(context.Background(), &s3.PutObjectInput{
    //         Bucket: aws.String("nowplaying"),
    //         Key: aws.String("database.db"),
    //         Body: bckup,
    //     })
    //
    //     if err != nil {
    //         log.Fatal(err.Error())
    //     }
    // }
    //
    log.Println("Exiting nowplaying safely")
}
