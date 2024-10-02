package argon2id

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2id struct {
    Memory uint32
    Iterations uint32
    Parrallel uint8
    SaltSize uint32
    KeySize uint32
    Key []byte
    Salt []byte
}

func getBytes(n uint32) ([]byte, error) {
    b := make([]byte, n)

    _, err := rand.Read(b)

    return b, err
}


func NewArgon2id(m uint32, i uint32, p uint8, s uint32, k uint32) *Argon2id {
    return &Argon2id{
        Memory: m,
        Iterations: i,
        Parrallel: p,
        SaltSize: s,
        KeySize: k,
    }
}

func (a *Argon2id) GenerateFromString(value string) ([]byte, error) {
    salt := a.Salt

    if salt == nil {
        var err error

        salt, err = getBytes(a.SaltSize)
        if err != nil {
            return nil, err
        }
    }

    hash := argon2.IDKey([]byte(value), salt, a.Iterations, a.Memory, a.Parrallel, a.KeySize)
    return hash, nil
}

func (a *Argon2id) EncodeFromString(value string) (string, error) {
    salt, err := getBytes(a.SaltSize)
    if err != nil {
        return "", err
    }

    hash := argon2.IDKey([]byte(value), salt, a.Iterations, a.Memory, a.Parrallel, a.KeySize)

    encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
    encodedHash := base64.RawStdEncoding.EncodeToString(hash)

    return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, a.Memory, a.Iterations, a.Parrallel, encodedSalt, encodedHash), nil
}

func (a *Argon2id) Compare(value1, value2 string) (bool, error) {
    decodedArgon, err := a.DecodeToArgon2id(value2)
    if err != nil {
        return false, err
    }

    a.Salt = decodedArgon.Salt
    newHash, err := a.GenerateFromString(value1)
    if err != nil {
        return false, err
    }

    oldHash := decodedArgon.Key
    if subtle.ConstantTimeCompare(newHash, oldHash) == 1 {
        return true, nil
    }

    return false, nil
}

func (a *Argon2id) DecodeToArgon2id(encoded string) (*Argon2id, error) {
    vals := strings.Split(encoded, "$")
    if len(vals) != 6 {
        return nil, fmt.Errorf("Invalid Hash")
    }

    var version int
    _, err := fmt.Sscanf(vals[2], "v=%d", &version)
    if err != nil {
        return nil, err
    }
    if version != argon2.Version {
        return nil, fmt.Errorf("Incorrect Argon Version")
    }

    newArgon := &Argon2id{}
    _, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &newArgon.Memory, &newArgon.Iterations, &newArgon.Parrallel)
    if err != nil {
        return nil, err
    }

    salt, err := base64.RawStdEncoding.DecodeString(vals[4])
    if err != nil {
        return nil, err
    }

    newArgon.SaltSize = uint32(len(salt))
    newArgon.Salt = salt

    hash, err := base64.RawStdEncoding.DecodeString(vals[5])
    if err != nil {
        return nil, err
    }

    newArgon.KeySize = uint32(len(hash))
    newArgon.Key = hash

    return newArgon, nil
}
