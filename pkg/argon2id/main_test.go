package argon2id

import (
	"testing"
)

func TestArgon2id(t *testing.T) {
    a := NewArgon2id(16 * 1024, 2, 2, 16, 32)

    hash, err := a.EncodeFromString("password")

    if err != nil {
        t.Fatalf("Oops: %s\n", err)
    }

    t.Logf("Hash: %s\n", hash)

    b := NewArgon2id(16 * 1024, 2, 2, 16, 32)
    pass, err := b.Compare("password", hash)

    if err != nil {
        t.Fatalf("Oops: %s\n", err)
    }

    if !pass {
        t.Fatalf("Compare Fail: %v", pass)
    }
}
