package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"strings"
	"time"
)

type SongMetadata struct {
    Name string
    Track string
    Image string
}

func return500(w http.ResponseWriter) {
    encode(w, 500, ResponseError{ Success: false, Messaage: INTERNAL_ERROR, Code: INTERNAL_SERVER_ERROR })
}

func encode[T any](w http.ResponseWriter, status int, v T) error {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(v); err != nil {
        return fmt.Errorf("encoding: %w", err)
    }

    return nil
}

func decode[T any](r *http.Request) (T, error) {
    var v T
    if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
        return v, fmt.Errorf("decoding: %w", err)
    }

    return v, nil
}

func loadArtistImages(metadata []Artist, cfg Config) {
    type DiscogResp struct {
        Results []struct {
            Title string `json:"title"`
            Thumb string `json:"thumb"`
        } `json:"results"`
    }

    client := http.Client{
        Timeout: time.Millisecond * 600,
    }

    results := make(chan struct{})

    for i, m := range metadata {
        go func(m Artist) {
            req, _ := http.NewRequest("GET", "https://api.discogs.com/database/search", nil)
            params := url.Values{}
            params.Set("key", cfg.Discogs.Key)
            params.Set("secret", cfg.Discogs.Secret)
            params.Set("q", strings.Trim(strings.Split(m.Name, ",")[0], " "))
            params.Set("type", "artist")
            req.Header.Set("User-Agent", "nowplayingapp 0.1 / mentemusic.com")
            req.URL.RawQuery = params.Encode()
            res, err := client.Do(req)

            if err != nil {
                log.Printf("Error Loading Artist Image: %s, %s", m.Name, err)
                results <- struct{}{}
                return
            }

            defer res.Body.Close()

            var data DiscogResp

            if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
                log.Printf("decoding: %s\n", err)
                results <- struct{}{}
                return
            }

            if len(data.Results) > 0 {
                metadata[i].Image = data.Results[0].Thumb
            } else {
                metadata[i].Image = ""
            }

            results <- struct{}{}
        }(m)
    }

    for range metadata {
        <- results
    }
}

func loadTrackImages(metadata []Track, cfg Config) {
    type DiscogResp struct {
        Results []struct {
            Title string `json:"title"`
            Thumb string `json:"thumb"`
        } `json:"results"`
    }

    client := http.Client{
        Timeout: time.Millisecond * 600,
    } 

    results := make(chan struct{})

    for i, m := range metadata {
        go func(m Track) {
            req, _ := http.NewRequest("GET", "https://api.discogs.com/database/search", nil)
            params := url.Values{}
            params.Set("key", cfg.Discogs.Key)
            params.Set("secret", cfg.Discogs.Secret)
            params.Set("q", fmt.Sprintf("%s - %s", m.Name, m.Track))
            req.Header.Set("User-Agent", "nowplayingapp 0.1 / mentemusic.com")
            req.URL.RawQuery = params.Encode()
            res, err := client.Do(req)

            if err != nil {
                log.Printf("Error Loading Images: %s, %s, %s", m.Name, m.Track, err)
                results <- struct{}{}
                return
            }

            defer res.Body.Close()

            var data DiscogResp

            if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
                log.Printf("decoding: %s\n", err)
                results <- struct{}{}
                return
            }

            if len(data.Results) > 0 {
                metadata[i].Image = data.Results[0].Thumb
            } else {
                metadata[i].Image = ""
            }

            results <- struct{}{}
        }(m)
    }

    for range metadata {
        <- results
    }
}
