package auth_test

import (
	"testing"

	"github.com/gatheryourdeals/data/internal/auth"
)

func TestHashPassword(t *testing.T) {
	hash, err := auth.HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "mypassword" {
		t.Fatal("hash should not equal the plaintext password")
	}
}

func TestHashPassword_DifferentEachTime(t *testing.T) {
	h1, _ := auth.HashPassword("same")
	h2, _ := auth.HashPassword("same")
	if h1 == h2 {
		t.Fatal("expected different hashes for the same password (bcrypt uses random salt)")
	}
}

func TestCheckPassword_Correct(t *testing.T) {
	hash, _ := auth.HashPassword("mypassword")

	err := auth.CheckPassword("mypassword", hash)
	if err != nil {
		t.Fatalf("expected nil for correct password, got %v", err)
	}
}

func TestCheckPassword_Wrong(t *testing.T) {
	hash, _ := auth.HashPassword("mypassword")

	err := auth.CheckPassword("wrongpassword", hash)
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}
