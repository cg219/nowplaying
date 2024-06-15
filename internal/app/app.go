package app

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"time"
)

type Config struct {
    LastFM struct {
        Key string `yaml:"key"`
        Secret string `yaml:"secret"`
    } `yaml:"lastfm"`
}

type tokenResp struct {
    Token string `json:"token"`
}

type apiParam struct {
    Name string
    Value string
}

type authCfg struct {
    key string
    secret string
    client *http.Client
    ctx context.Context
    token string
}

func makeSignature (secret string, list []apiParam) string {
    rawSig := ""

    sort.Slice(list, func(i, j int) bool {
        return list[i].Name < list[j].Name
    })

    for _, p := range(list) {
        rawSig = fmt.Sprintf("%s%s%s", rawSig, p.Name, p.Value)
    }

    rawSig = fmt.Sprintf("%s%s", rawSig, secret)
    h := md5.New()
    fmt.Fprint(h, rawSig)
    sig := h.Sum(nil)

    log.Print("Sig: ", rawSig)

    return fmt.Sprintf("%x", sig[:])
}

func (cfg *authCfg) makeApiUrl(method string, list []apiParam) string {
    baseurl := "http://ws.audioscrobbler.com/2.0/?format=json&api_sig=%s%s"
    params := ""

    list = append(list, apiParam{ Name: "api_key", Value: cfg.key })
    list = append(list, apiParam{ Name: "method", Value: method })

    for _, p := range(list) {
        params = fmt.Sprintf("%s&%s=%s", params, p.Name, p.Value)
    }

    log.Printf(baseurl, makeSignature(cfg.secret, list), params)
    return fmt.Sprintf(baseurl, makeSignature(cfg.secret, list), params)
}

func Auth(cfg *authCfg) error {
    authurl := "http://www.last.fm/api/auth/?api_key=%s&token=%s"

    respBody := tokenResp{}

    req, err := http.NewRequestWithContext(cfg.ctx, "GET", cfg.makeApiUrl("auth.gettoken", nil), nil)

    if err != nil {
        return err
    }

    resp, err := cfg.client.Do(req)

    if err != nil {
        return err
    }

    defer resp.Body.Close()
    
    err = json.NewDecoder(resp.Body).Decode(&respBody)

    if err != nil {
        return err
    }

    log.Print("Token: ", respBody.Token)

    exec.Command("open", fmt.Sprintf(authurl, cfg.key, respBody.Token)).Run()

    fmt.Println("Hit Enter to Continue after authorization")
    fmt.Scanln()

    req, err = http.NewRequestWithContext(cfg.ctx, "GET", cfg.makeApiUrl("auth.getsession", []apiParam{{ Name: "token", Value: respBody.Token }}), nil)

    if err != nil {
        return err
    }

    resp2, err := cfg.client.Do(req)

    if err != nil {
        return err
    }

    defer resp2.Body.Close()
    
    var tst map[string]interface{}

    err = json.NewDecoder(resp2.Body).Decode(&tst)

    if err != nil {
        return err
    }

    log.Print(tst)
    return nil
}

func Run(config Config) error {
    cfg := &authCfg{
        key: config.LastFM.Key, 
        secret: config.LastFM.Secret,
        client: &http.Client{
            Timeout: time.Second * 60,
        },
        ctx: context.Background(),
    }

    if err := Auth(cfg); err != nil {
        return err
    }

    return nil
}
