package models

import (
	"testing"

	"golang.org/x/crypto/nacl/secretbox"
)

var userTokenKey = []byte(`I3w8GGTsb9R3SKCvRzUd4aNasYIhX2IC`)

func TestBuild_EncryptAndSetGitHubCredential(t *testing.T) {
	b := Build{}
	var key [32]byte
	copy(key[:], userTokenKey)
	if err := b.EncryptAndSetGitHubCredential([]byte(`asdf`), key); err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	// msg size + nonce + overhead
	if n := len(b.EncryptedGitHubCredential); n != 4+24+secretbox.Overhead {
		t.Fatalf("bad size: %v", n)
	}
}

func TestBuild_GetGitHubCredential(t *testing.T) {
	b := Build{}
	var key [32]byte
	copy(key[:], userTokenKey)
	if err := b.EncryptAndSetGitHubCredential([]byte(`asdf`), key); err != nil {
		t.Fatalf("encrypt should have succeeded: %v", err)
	}
	cred, err := b.GetGitHubCredential(key)
	if err != nil {
		t.Fatalf("should have succeeded: %v", err)
	}
	if cred != "asdf" {
		t.Fatalf("unexpected credential: %v", string(cred))
	}
}
