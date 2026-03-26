package auth_test

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/gatheryourdeals/data/internal/auth"
)

func TestMain(m *testing.M) {
	auth.BcryptCost = bcrypt.MinCost
	os.Exit(m.Run())
}
