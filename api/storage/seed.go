package storage

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
)

// SeedDirectory uploads the demo images under dir, skipping those already stored.
// The installation's files are dir/<folder>/<file>, stored under that path. A
// council's files are the references listed in dir/refs.txt (written by the
// seed generator, one "<council id>/<folder>/<file>" per line), each read from
// dir/<folder>/<file>: the same demo picture goes to every council that uses it.
func SeedDirectory(ctx context.Context, dir string) (uploaded int, skipped int, err error) {
	var s *service
	if s, err = client(); err != nil {
		return 0, 0, err
	}
	var seed = func(ref string, path string) error {
		var contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
		if !ValidRef(ref) || !strings.HasPrefix(contentType, "image/") {
			return nil // credits files and anything outside the known folders
		}
		var bucket, key = s.location(ref)
		if _, statErr := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{}); statErr == nil {
			skipped++
			return nil
		}
		var file, openErr = os.Open(path)
		if openErr != nil {
			return openErr
		}
		defer file.Close()
		var info, _ = file.Stat()
		if putErr := Put(ctx, ref, file, info.Size(), contentType); putErr != nil {
			return putErr
		}
		uploaded++
		return nil
	}

	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		var rel, relErr = filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		var ref = filepath.ToSlash(rel)
		if folder, _, _ := strings.Cut(ref, "/"); !PlatformFolders[folder] {
			return nil
		}
		return seed(ref, path)
	})
	if err != nil {
		return uploaded, skipped, fmt.Errorf("seeding %s: %w", dir, err)
	}

	var refs *os.File
	refs, err = os.Open(filepath.Join(dir, "refs.txt"))
	if errors.Is(err, fs.ErrNotExist) {
		return uploaded, skipped, nil
	}
	if err != nil {
		return uploaded, skipped, err
	}
	defer refs.Close()
	var lines = bufio.NewScanner(refs)
	for lines.Scan() {
		var ref = strings.TrimSpace(lines.Text())
		var _, source, _ = strings.Cut(ref, "/") // "<folder>/<file>" under dir
		if ref == "" || !ValidRef(ref) {
			continue
		}
		if err = seed(ref, filepath.Join(dir, filepath.FromSlash(source))); err != nil {
			return uploaded, skipped, fmt.Errorf("seeding %s: %w", ref, err)
		}
	}
	return uploaded, skipped, lines.Err()
}
