package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/yaml.v3"
)

type Config struct {
    Data struct {
        Path string `yaml:"data"`
    } `yaml:"app"`
    R2 struct {
        Key string `yaml:"key"`
        Secret string `yaml:"secret"`
        Token string `yaml:"token"`
        Url string `yaml:"url"`
    } `yaml:"r2"`
}

func NewConfig() *Config {
    cfg := &Config{}

    cfg.R2.Key = os.Getenv("R2_KEY")
    cfg.R2.Secret = os.Getenv("R2_SECRET")
    cfg.R2.Token = os.Getenv("R2_TOKEN")
    cfg.R2.Url = os.Getenv("R2_URL")
    cfg.Data.Path = os.Getenv("APP_DATA")

    return cfg
}

func NewConfigFromSecrets(data []byte) *Config {
    cfg := &Config{}

    if err := yaml.Unmarshal(data, cfg); err != nil {
        log.Fatal("Error unmarshalling secrets file")
    }

    return cfg
}

func Run(cfg *Config, client *s3.Client) {
    cwd, _ := os.Getwd();
    dbfile, err := os.Open(filepath.Join(cwd, cfg.Data.Path))

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

func main() {
    var cfg *Config

    secretsPath := os.Getenv("NP_CREDTENTIALS")
    _, err := os.Stat(secretsPath)

    if err != nil {
        if os.IsNotExist(err) {
            log.Printf("secrets file not found: %s\nFalling back to env variables\n", secretsPath)
            cfg = NewConfig()
        } else if os.IsPermission(err) {
            log.Printf("incorrect permissions on secret file: %s\nFalling back to env variables\n", secretsPath)
            cfg = NewConfig()
        } else {
            log.Fatal(err)
        }
    } else {
        data, err := os.ReadFile(secretsPath)
        if err != nil {
            log.Printf("error loading secrets file: %s; err: %s\nFalling back to env variables\n", secretsPath, err.Error())
        }

        cfg = NewConfigFromSecrets(data)
    }

    s3cfg, err := config.LoadDefaultConfig(context.Background(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.R2.Key, cfg.R2.Secret, "")), config.WithRegion("auto"))

    if err != nil {
        log.Fatal(err)
    }

    client := s3.NewFromConfig(s3cfg, func(o *s3.Options) {
        o.BaseEndpoint = aws.String(cfg.R2.Url)
    })

    ticker := time.NewTicker(time.Hour * 4)
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    prog:
    for {
        select {
        case <- ticker.C:
            Run(cfg, client)
        case <- ctx.Done():
            ticker.Stop()
            break prog
        }
    }

    if strings.EqualFold(os.Getenv("APP_EXIT_BACKUP"), "1") {
        Run(cfg, client)
    }

    log.Println("Exiting nowplaying safely")
}

