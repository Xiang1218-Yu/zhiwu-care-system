package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mime/multipart"
)

func TestSaveUploadValidatesImageContent(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "source.png")
	if err := os.WriteFile(path, []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	}, 0600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	uploadURL, err := SaveUpload(file, &multipart.FileHeader{Filename: "leaf.png", Size: 16}, directory)
	if err != nil {
		t.Fatalf("expected valid png upload, got %v", err)
	}
	if !strings.HasPrefix(uploadURL, "/uploads/") {
		t.Fatalf("unexpected upload URL: %s", uploadURL)
	}
	if _, err := os.Stat(filepath.Join(directory, filepath.Base(uploadURL))); err != nil {
		t.Fatalf("saved file not found: %v", err)
	}
}

func TestSaveUploadRejectsOversizedFile(t *testing.T) {
	file, err := os.Open(filepath.Join(t.TempDir(), "empty"))
	if err == nil {
		file.Close()
		t.Fatal("expected missing source file")
	}

	directory := t.TempDir()
	path := filepath.Join(directory, "source.jpg")
	if err := os.WriteFile(path, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	file, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = SaveUpload(file, &multipart.FileHeader{Filename: "large.jpg", Size: MaxUploadSize + 1}, directory)
	if err == nil || !strings.Contains(err.Error(), "8MB") {
		t.Fatalf("expected size validation error, got %v", err)
	}
}
