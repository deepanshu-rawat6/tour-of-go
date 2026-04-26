package image

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	registryBase = "https://registry-1.docker.io/v2"
	authBase     = "https://auth.docker.io/token"
)

// Pull fetches an image from Docker Hub and extracts it into the store.
func Pull(store *Store, name, tag string) error {
	fmt.Printf("Pulling %s:%s...\n", name, tag)

	token, err := fetchToken(name)
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	manifest, err := fetchManifest(name, tag, token)
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}

	rootfs := store.RootfsDir(name, tag)
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return err
	}

	for i, layer := range manifest.Layers {
		fmt.Printf("Downloading layer %d/%d (%s)...\n", i+1, len(manifest.Layers), layer.Digest[:19])
		if err := pullLayer(name, layer.Digest, token, rootfs); err != nil {
			return fmt.Errorf("layer %s: %w", layer.Digest[:12], err)
		}
	}

	fmt.Printf("Done. Image stored at %s\n", store.ImageDir(name, tag))
	return nil
}

// fetchToken gets an anonymous Bearer token for pulling public images.
func fetchToken(name string) (string, error) {
	// Docker Hub library images use "library/<name>" scope
	repo := name
	if !strings.Contains(name, "/") {
		repo = "library/" + name
	}
	url := fmt.Sprintf("%s?service=registry.docker.io&scope=repository:%s:pull", authBase, repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

type manifest struct {
	Layers []struct {
		Digest string `json:"digest"`
	} `json:"layers"`
}

// fetchManifest retrieves the image manifest (schema v2).
func fetchManifest(name, tag, token string) (*manifest, error) {
	repo := name
	if !strings.Contains(name, "/") {
		repo = "library/" + name
	}
	url := fmt.Sprintf("%s/%s/manifests/%s", registryBase, repo, tag)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned %s", resp.Status)
	}
	var m manifest
	return &m, json.NewDecoder(resp.Body).Decode(&m)
}

// pullLayer downloads a single layer blob and extracts it into destDir.
func pullLayer(name, digest, token, destDir string) error {
	repo := name
	if !strings.Contains(name, "/") {
		repo = "library/" + name
	}
	url := fmt.Sprintf("%s/%s/blobs/%s", registryBase, repo, digest)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("blob fetch returned %s", resp.Status)
	}
	return extractTar(resp.Body, destDir)
}

// extractTar extracts a gzipped tar stream into destDir.
func extractTar(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Skip whiteout files (Docker layer deletions)
		base := filepath.Base(hdr.Name)
		if strings.HasPrefix(base, ".wh.") {
			target := filepath.Join(destDir, filepath.Dir(hdr.Name), strings.TrimPrefix(base, ".wh."))
			os.RemoveAll(target)
			continue
		}

		target := filepath.Join(destDir, hdr.Name)
		// Prevent path traversal
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, os.FileMode(hdr.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			io.Copy(f, tr)
			f.Close()
		case tar.TypeSymlink:
			os.MkdirAll(filepath.Dir(target), 0755)
			os.Symlink(hdr.Linkname, target)
		case tar.TypeLink:
			os.MkdirAll(filepath.Dir(target), 0755)
			os.Link(filepath.Join(destDir, hdr.Linkname), target)
		}
	}
	return nil
}
