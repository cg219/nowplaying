package webtoken

import (
	"cmp"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Token struct {
    subject string
    expiresAt time.Time
    Name string
    value string
    secret string
}

type Subject struct {
    Value string `json:"value"`
    RefreshToken string `json:"refresh"`
}

func NewToken(name, subject, secret string, expires time.Time) Token {
    return Token{
        Name: name,
        subject: subject,
        expiresAt: expires,
        secret: secret,
    }
}

func NewCookie(name, value, path string, maxage int) http.Cookie {
    return http.Cookie{
        Name: fmt.Sprintf("%s", name),
        Value: value,
        Path: path,
        MaxAge: maxage,
        HttpOnly: true,
        Secure: true,
        SameSite: http.SameSiteLaxMode,
    }
}

func (t *Token) Value() string {
    return t.value
}

func (t *Token) Secret() string {
    return t.secret
}

// TODO: Add ability to pass custom claims
func GetParsedJWT(value string, secret string) (*jwt.Token, error) {
    token, err := jwt.ParseWithClaims(value, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) { return []byte(secret), nil })
    if err != nil {
        return nil, fmt.Errorf("Parsing JWT")
    }

    return token, nil
}

// TODO: Add ability to pass custom claims
func (t *Token) Create(issuer string) error {
    salt := make([]byte, 16)
    hash := sha256.New()
    now := fmt.Sprintf("%d", time.Now().UnixMilli())

    rand.Read(salt)
    hash.Write([]byte(t.subject))
    hash.Write([]byte(issuer))
    hash.Write([]byte(now))
    hash.Write(salt)

    encodedSubject, err := json.Marshal(&Subject{
        Value: t.subject,
        RefreshToken: string(hash.Sum(nil)),
    })

    if err != nil {
        return fmt.Errorf("Error creating %s token. val: %t, err %s", cmp.Or(t.Name, "new"), t.subject, err)
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.RegisteredClaims{
        Issuer: issuer,
        IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
        ExpiresAt: jwt.NewNumericDate(t.expiresAt),
        Subject: string(encodedSubject),
    })

    stoken, err := token.SignedString([]byte(t.secret))

    if err != nil {
        return fmt.Errorf("Error creating %s token. val: %t, err %s", cmp.Or(t.Name, "new"), t.subject, err)
    }

    t.value = stoken

    return  nil
}

