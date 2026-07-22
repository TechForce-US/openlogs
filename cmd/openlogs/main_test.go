package main

import (
	"path/filepath"
	"testing"

	"github.com/jmstewart1127/openlogs/internal/db"
)

func testDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestBootstrapAdminNoop(t *testing.T) {
	d := testDB(t)
	// Neither / partial env → no user created, no error.
	for _, tc := range []struct{ email, pass string }{
		{"", ""},
		{"admin@example.com", ""},
		{"", "supersecret"},
	} {
		if err := bootstrapAdmin(d, tc.email, tc.pass); err != nil {
			t.Fatalf("bootstrapAdmin(%q,%q): %v", tc.email, tc.pass, err)
		}
	}
	if u, _ := d.GetUserByEmail("admin@example.com"); u != nil {
		t.Fatal("no user should have been created")
	}
}

func TestBootstrapAdminCreatesAndIsIdempotent(t *testing.T) {
	d := testDB(t)

	if err := bootstrapAdmin(d, "admin@example.com", "supersecret"); err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	u, err := d.GetUserByEmail("admin@example.com")
	if err != nil || u == nil {
		t.Fatalf("expected user created, got %v (err %v)", u, err)
	}
	firstHash := u.PasswordHash

	// Running again is a no-op and must not change the existing user/password.
	if err := bootstrapAdmin(d, "admin@example.com", "a-different-password"); err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
	u2, _ := d.GetUserByEmail("admin@example.com")
	if u2.PasswordHash != firstHash {
		t.Fatal("idempotent bootstrap must not overwrite an existing user's password")
	}
}

func TestBootstrapAdminRejectsShortPassword(t *testing.T) {
	d := testDB(t)
	if err := bootstrapAdmin(d, "admin@example.com", "short"); err == nil {
		t.Fatal("expected error for password < 8 chars")
	}
	if u, _ := d.GetUserByEmail("admin@example.com"); u != nil {
		t.Fatal("no user should be created when password is rejected")
	}
}
