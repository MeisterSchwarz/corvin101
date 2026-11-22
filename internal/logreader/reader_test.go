package logreader

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWatchDetectsSameFileRewriteWithoutShrink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.log")
	original := "old state\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	lines := make(chan string, 10)
	resets := make(chan struct{}, 10)
	done := make(chan error, 1)
	r := New(path).FromOffset(int64(len(original))).OnReset(func() { resets <- struct{}{} })
	go func() { done <- r.Watch(ctx, func(s string) { lines <- s }) }()
	// Establish that watching began and normal append does not reset state.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("appended\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-lines:
		if got != "appended\n" {
			t.Fatal(got)
		}
	case <-ctx.Done():
		t.Fatal("append missed")
	}
	select {
	case <-resets:
		t.Fatal("ordinary append reset state")
	default:
	}
	// Overwrite in place with a longer stream, keeping the same file identity.
	f, err = os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteAt([]byte("new zone\nnew battle\n"+strings.Repeat("padding", 10)+"\n"), 0)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-resets:
	case <-ctx.Done():
		t.Fatal("rewrite not detected")
	}
	select {
	case got := <-lines:
		if got != "new zone\n" {
			t.Fatalf("replay began at stale offset: %q", got)
		}
	case <-ctx.Done():
		t.Fatal("replay missed")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("watch did not stop")
	}
}
