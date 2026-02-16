package blob

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestFilesystemStore_PutGetDelete(t *testing.T) {
	root := filepath.Join(t.TempDir(), "blobs")
	store, err := NewFilesystemStore(root)
	if err != nil {
		t.Fatalf("new filesystem store: %v", err)
	}

	key := "inbound/attachments/1/2/file.txt"
	payload := []byte("hello")
	if err := store.Put(context.Background(), key, "text/plain", payload); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, err := store.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("unexpected payload: %q", string(got))
	}

	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = store.Get(context.Background(), key)
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}
}
