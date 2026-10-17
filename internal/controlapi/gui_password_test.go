package controlapi

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

func TestCreateGUIPasswordWrites0600BcryptFile(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateGUIPassword("correct horse"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(store.Dir(), "gui-password"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o", info.Mode().Perm())
	}
	ok, err := store.PasswordMatches("correct horse")
	if err != nil || !ok {
		t.Fatalf("match = %v, %v", ok, err)
	}
	ok, err = store.PasswordMatches("wrong")
	if err != nil || ok {
		t.Fatalf("wrong password matched: %v %v", ok, err)
	}
}

func TestCreateGUIPasswordRejectsShortAndDuplicate(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateGUIPassword("short"); err == nil {
		t.Fatal("expected short password error")
	}
	if err := store.CreateGUIPassword("long-enough-password"); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateGUIPassword("another-long-password"); !errors.Is(err, errGUIPasswordConfigured) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateGUIPasswordConcurrentOnlyOneSucceeds(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.CreateGUIPassword("long-enough-password")
		}()
	}
	wg.Wait()
	close(errs)
	var ok, dup int
	for err := range errs {
		if err == nil {
			ok++
		} else if errors.Is(err, errGUIPasswordConfigured) {
			dup++
		} else {
			t.Fatalf("unexpected %v", err)
		}
	}
	if ok != 1 || dup != 7 {
		t.Fatalf("ok=%d dup=%d", ok, dup)
	}
}

func TestReplaceGUIPasswordChangesHash(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateGUIPassword("old-password-ok"); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceGUIPassword("new-password-ok"); err != nil {
		t.Fatal(err)
	}
	ok, err := store.PasswordMatches("new-password-ok")
	if err != nil || !ok {
		t.Fatal("new password should match")
	}
	ok, err = store.PasswordMatches("old-password-ok")
	if err != nil || ok {
		t.Fatalf("old password should not match: %v %v", ok, err)
	}
}

func TestPasswordMatchesMissingFile(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	ok, err := store.PasswordMatches("anything1")
	if ok || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if store.GUIPasswordConfigured() {
		t.Fatal("missing file must not count as configured")
	}
}

func TestGUIPasswordRejectsTooLong(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	long := make([]rune, 129)
	for i := range long {
		long[i] = 'a'
	}
	if utf8.RuneCountInString(string(long)) != 129 {
		t.Fatal("fixture")
	}
	if err := store.CreateGUIPassword(string(long)); err == nil {
		t.Fatal("expected too long")
	}
}

func TestCreateGUIPasswordAcceptsMoreThan72Bytes(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	password := strings.Repeat("a", 73)
	if err := store.CreateGUIPassword(password); err != nil {
		t.Fatal(err)
	}
	ok, err := store.PasswordMatches(password)
	if err != nil || !ok {
		t.Fatalf("match = %v, %v", ok, err)
	}
}

func TestGUIPasswordConfiguredWhenFileUnusable(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir(), "gui-password")
	if err := os.WriteFile(path, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !store.GUIPasswordConfigured() {
		t.Fatal("existing unusable file must still count as configured")
	}
	ok, err := store.PasswordMatches("long-enough-password")
	if ok || err == nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatal("unusable file must not look missing")
	}
}
