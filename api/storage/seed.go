package storage

import (
	"context"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
)

// SeedDirectory uploads the demo images under dir (dir/<folder>/<file>) with their path as
// reference, skipping those already stored. Only image files in known folders are sent.
func SeedDirectory(ctx context.Context, dir string) (uploaded int, skipped int, err error) {
	var s *service
	if s, err = client(); err != nil {
		return 0, 0, err
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
		var contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(ref)))
		if !ValidRef(ref) || !strings.HasPrefix(contentType, "image/") {
			return nil // credits files and anything outside the known folders
		}
		if _, statErr := s.client.StatObject(ctx, s.bucket, ref, minio.StatObjectOptions{}); statErr == nil {
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
	})
	if err != nil {
		return uploaded, skipped, fmt.Errorf("seeding %s: %w", dir, err)
	}
	return uploaded, skipped, nil
}
