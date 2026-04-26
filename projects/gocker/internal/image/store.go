package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store manages the local image cache at ~/.gocker/images/<name>:<tag>/
type Store struct {
	Base string // e.g. ~/.gocker
}

func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	base := filepath.Join(home, ".gocker")
	if err := os.MkdirAll(filepath.Join(base, "images"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(base, "containers"), 0755); err != nil {
		return nil, err
	}
	return &Store{Base: base}, nil
}

func (s *Store) ImageDir(name, tag string) string {
	return filepath.Join(s.Base, "images", name+":"+tag)
}

func (s *Store) RootfsDir(name, tag string) string {
	return filepath.Join(s.ImageDir(name, tag), "rootfs")
}

func (s *Store) ContainerDir(id string) string {
	return filepath.Join(s.Base, "containers", id)
}

// Exists reports whether the image rootfs has been pulled.
func (s *Store) Exists(name, tag string) bool {
	_, err := os.Stat(s.RootfsDir(name, tag))
	return err == nil
}

// List returns all cached images as "name:tag" strings.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.Base, "images"))
	if err != nil {
		return nil, err
	}
	var images []string
	for _, e := range entries {
		if e.IsDir() {
			images = append(images, e.Name())
		}
	}
	return images, nil
}

// ParseRef splits "alpine:3.19" into ("alpine", "3.19").
// Defaults tag to "latest" if omitted.
func ParseRef(ref string) (name, tag string) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], "latest"
}

// DirSize returns the total size of a directory tree in bytes.
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func FormatSize(b int64) string {
	const mb = 1024 * 1024
	if b >= mb {
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	}
	return fmt.Sprintf("%d KB", b/1024)
}
