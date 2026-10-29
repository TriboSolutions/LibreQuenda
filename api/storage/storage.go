// Package storage keeps uploaded files in Garage (self-hosted, S3-compatible).
//
// The database only ever stores references (object keys such as
// "councils/01a1….webp"). A reference becomes a URL only in API responses,
// through URL(), so the storage backend or the public path can change without
// touching any stored data.
package storage

import (
	"context"
	"fmt"
	"io"
	"librequenda/shared"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// The public path the API serves stored files under (see core/media).
const publicPath = "/api/media/"

type Object struct {
	Reader       io.ReadCloser
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

type service struct {
	client *minio.Client
	bucket string
}

var instance *service
var initErr error
var once sync.Once

// Client connects to the storage on first use.
func client() (*service, error) {
	once.Do(func() {
		for name, value := range map[string]string{
			"STORAGE_ENDPOINT":   shared.STORAGE_ENDPOINT,
			"STORAGE_BUCKET":     shared.STORAGE_BUCKET,
			"STORAGE_ACCESS_KEY": shared.STORAGE_ACCESS_KEY,
			"STORAGE_SECRET_KEY": shared.STORAGE_SECRET_KEY,
		} {
			if value == "" {
				initErr = fmt.Errorf("%s environment variable is not set", name)
				return
			}
		}
		var c, err = minio.New(shared.STORAGE_ENDPOINT, &minio.Options{
			Creds:        credentials.NewStaticV4(shared.STORAGE_ACCESS_KEY, shared.STORAGE_SECRET_KEY, ""),
			Secure:       shared.STORAGE_USE_SSL,
			Region:       shared.STORAGE_REGION,
			BucketLookup: minio.BucketLookupPath, // Garage on the internal network: no bucket subdomains
		})
		if err != nil {
			initErr = fmt.Errorf("failed to create storage client: %w", err)
			return
		}
		instance = &service{client: c, bucket: shared.STORAGE_BUCKET}
	})
	return instance, initErr
}

// Put stores a file under the reference (object key) it is given.
func Put(ctx context.Context, ref string, reader io.Reader, size int64, contentType string) error {
	var s, err = client()
	if err != nil {
		return err
	}
	_, err = s.client.PutObject(ctx, s.bucket, ref, reader, size, minio.PutObjectOptions{
		ContentType:  contentType,
		CacheControl: "public, max-age=31536000, immutable", // references are never reused
	})
	if err != nil {
		return fmt.Errorf("failed to store %q: %w", ref, err)
	}
	return nil
}

// Get opens a stored file; the caller closes Reader. Missing files return ErrNotFound.
func Get(ctx context.Context, ref string) (*Object, error) {
	var s, err = client()
	if err != nil {
		return nil, err
	}
	var object *minio.Object
	object, err = s.client.GetObject(ctx, s.bucket, ref, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	var info minio.ObjectInfo
	info, err = object.Stat()
	if err != nil {
		object.Close()
		if code := minio.ToErrorResponse(err).Code; code == "NoSuchKey" || code == "NotFound" {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &Object{Reader: object, Size: info.Size, ContentType: info.ContentType, LastModified: info.LastModified, ETag: info.ETag}, nil
}

// Delete removes a stored file; one already gone is not an error.
func Delete(ctx context.Context, ref string) error {
	var s, err = client()
	if err != nil {
		return err
	}
	if err = s.client.RemoveObject(ctx, s.bucket, ref, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete %q: %w", ref, err)
	}
	return nil
}

// Ready checks that the bucket can be reached.
func Ready(ctx context.Context) error {
	var s, err = client()
	if err != nil {
		return err
	}
	var exists bool
	exists, err = s.client.BucketExists(ctx, s.bucket)
	if err == nil && !exists {
		err = fmt.Errorf("bucket %q does not exist", s.bucket)
	}
	return err
}

var ErrNotFound = fmt.Errorf("not found")

// URL is the only place a reference becomes something to display. Empty in, empty out.
func URL(ref string) string {
	if ref == "" {
		return ""
	}
	return publicPath + ref
}

// ValidRef reports whether a reference has the shape this API creates: "<folder>/<name>.<ext>".
func ValidRef(ref string) bool {
	var folder, name, found = strings.Cut(ref, "/")
	return found && Folders[folder] && name != "" && !strings.ContainsAny(name, "/\\") && !strings.Contains(ref, "..")
}

// Folders that uploads may go to.
var Folders = map[string]bool{"branding": true, "councils": true, "facilities": true, "users": true}
