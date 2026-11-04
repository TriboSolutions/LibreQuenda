// Package storage keeps uploaded files in Garage (self-hosted, S3-compatible),
// in private buckets: one per council ("council-<id>") for its logo, its
// facilities' images and the photos of the problems reported there, and the
// installation's (STORAGE_BUCKET) for profile photos and the installation logo.
//
// The database only ever stores references, which say where the file is:
// "<council id>/<folder>/<name>" for a council's files, "<folder>/<name>" for
// the installation's. Nothing is public: a reference becomes something to
// display only in API responses, through URL(), as a presigned link that
// expires (like a private S3 bucket's). The proxy hands /storage/* to Garage.
package storage

import (
	"context"
	"fmt"
	"io"
	"librequenda/shared"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Where the proxy hands presigned requests to Garage (proxy/Caddyfile).
const publicPath = "/storage"

// How long a presigned link works (the S3 maximum is 7 days), and how soon a
// new one is signed: the same link in between, so browsers keep the image cached.
const (
	linkLifetime = 7 * 24 * time.Hour
	linkRenewal  = 3 * 24 * time.Hour
)

type Object struct {
	Reader       io.ReadCloser
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

type service struct {
	client    *minio.Client
	presigner *minio.Client // the same keys, signing for the public address
	bucket    string        // the installation's
	buckets   sync.Map      // council buckets known to exist
	links     sync.Map      // ref -> link
}

type link struct {
	url     string
	renewAt time.Time
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
		var public, parseErr = url.Parse(shared.PUBLIC_URL)
		if parseErr != nil || public.Host == "" {
			initErr = fmt.Errorf("PUBLIC_URL %q is not an absolute URL", shared.PUBLIC_URL)
			return
		}
		// signs only, never connects: with the region set it asks Garage nothing
		var presigner *minio.Client
		presigner, err = minio.New(public.Host, &minio.Options{
			Creds:        credentials.NewStaticV4(shared.STORAGE_ACCESS_KEY, shared.STORAGE_SECRET_KEY, ""),
			Secure:       public.Scheme == "https",
			Region:       shared.STORAGE_REGION,
			BucketLookup: minio.BucketLookupPath,
		})
		if err != nil {
			initErr = fmt.Errorf("failed to create storage presigner: %w", err)
			return
		}
		instance = &service{client: c, presigner: presigner, bucket: shared.STORAGE_BUCKET}
	})
	return instance, initErr
}

// Put stores a file under the reference (object key) it is given.
func Put(ctx context.Context, ref string, reader io.Reader, size int64, contentType string) error {
	var s, err = client()
	if err != nil {
		return err
	}
	var bucket, key = s.location(ref)
	if err = s.ensureBucket(ctx, bucket); err != nil {
		return err
	}
	_, err = s.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
		ContentType:  contentType,
		CacheControl: "private, max-age=604800, immutable", // references are never reused; links last a week
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
	var bucket, key = s.location(ref)
	var object *minio.Object
	object, err = s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	var info minio.ObjectInfo
	info, err = object.Stat()
	if err != nil {
		object.Close()
		if code := minio.ToErrorResponse(err).Code; code == "NoSuchKey" || code == "NotFound" || code == "NoSuchBucket" {
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
	var bucket, key = s.location(ref)
	err = s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	if err != nil && minio.ToErrorResponse(err).Code != "NoSuchBucket" {
		return fmt.Errorf("failed to delete %q: %w", ref, err)
	}
	s.links.Delete(ref)
	return nil
}

// DeleteCouncil removes a council's bucket with everything in it, when the
// council itself is deleted. A council that never stored a file has none.
func DeleteCouncil(ctx context.Context, council uuid.UUID) error {
	var s, err = client()
	if err != nil {
		return err
	}
	var bucket = councilBucket(council)
	var objects = make(chan minio.ObjectInfo)
	go func() {
		defer close(objects)
		for object := range s.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				return
			}
			objects <- object
		}
	}()
	for removeErr := range s.client.RemoveObjects(ctx, bucket, objects, minio.RemoveObjectsOptions{}) {
		if removeErr.Err != nil {
			return fmt.Errorf("failed to empty %s: %w", bucket, removeErr.Err)
		}
	}
	err = s.client.RemoveBucket(ctx, bucket)
	if err != nil && minio.ToErrorResponse(err).Code != "NoSuchBucket" {
		return fmt.Errorf("failed to delete %s: %w", bucket, err)
	}
	s.buckets.Delete(bucket)
	var prefix = council.String() + "/"
	s.links.Range(func(ref, _ any) bool {
		if strings.HasPrefix(ref.(string), prefix) {
			s.links.Delete(ref)
		}
		return true
	})
	return nil
}

// ensureBucket creates a council's bucket the first time it stores a file.
// Buckets are private: Garage serves nothing without a signature.
func (s *service) ensureBucket(ctx context.Context, bucket string) error {
	if bucket == s.bucket {
		return nil
	}
	if _, known := s.buckets.Load(bucket); known {
		return nil
	}
	var exists, err = s.client.BucketExists(ctx, bucket)
	if err == nil && !exists {
		err = s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: shared.STORAGE_REGION})
		if code := minio.ToErrorResponse(err).Code; code == "BucketAlreadyOwnedByYou" || code == "BucketAlreadyExists" {
			err = nil // created meanwhile by another upload
		}
	}
	if err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
	}
	s.buckets.Store(bucket, true)
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

// URL is the only place a reference becomes something to display: a presigned
// link to the private bucket, valid for a week. Empty in, empty out; a link
// that cannot be signed is empty too (the screens show no image).
func URL(ref string) string {
	if ref == "" || !ValidRef(ref) {
		return ""
	}
	var s, err = client()
	if err != nil {
		shared.Logger.Error("storage link for %s: %v", ref, err)
		return ""
	}
	var now = time.Now()
	if cached, ok := s.links.Load(ref); ok && now.Before(cached.(link).renewAt) {
		return cached.(link).url
	}
	var bucket, key = s.location(ref)
	var signed *url.URL
	signed, err = s.presigner.PresignedGetObject(context.Background(), bucket, key, linkLifetime, nil)
	if err != nil {
		shared.Logger.Error("storage link for %s: %v", ref, err)
		return ""
	}
	// signed for /<bucket>/<key>, which is what Garage sees once the proxy strips /storage
	signed.Path = publicPath + signed.Path
	signed.RawPath = ""
	var value = signed.String()
	s.links.Store(ref, link{url: value, renewAt: now.Add(linkRenewal)})
	return value
}

// Folders of a council's files, in its own bucket.
var CouncilFolders = map[string]bool{"councils": true, "facilities": true, "issues": true}

// Folders of the installation's files: profile photos, the installation's logo.
var PlatformFolders = map[string]bool{"branding": true, "users": true}

// CouncilRef is where a new file of a council goes.
func CouncilRef(council uuid.UUID, folder string, name string) string {
	return council.String() + "/" + folder + "/" + name
}

// ValidRef reports whether a reference has the shape this API creates:
// "<council id>/<council folder>/<name>" or "<platform folder>/<name>".
func ValidRef(ref string) bool {
	if strings.Contains(ref, "..") || strings.Contains(ref, "\\") {
		return false
	}
	var parts = strings.Split(ref, "/")
	switch len(parts) {
	case 2:
		return PlatformFolders[parts[0]] && parts[1] != ""
	case 3:
		var id, err = uuid.Parse(parts[0])
		return err == nil && id.String() == parts[0] && CouncilFolders[parts[1]] && parts[2] != ""
	}
	return false
}

// Owned reports whether a reference is a file of that council (uuid.Nil: of the
// installation) in that folder: what a form may save, so nobody links another
// council's files (and gets them signed) by sending their reference.
func Owned(ref string, council uuid.UUID, folder string) bool {
	if !ValidRef(ref) {
		return false
	}
	var parts = strings.Split(ref, "/")
	if council == uuid.Nil {
		return len(parts) == 2 && parts[0] == folder
	}
	return len(parts) == 3 && parts[0] == council.String() && parts[1] == folder
}

func councilBucket(council uuid.UUID) string {
	return "council-" + council.String()
}

// location: the bucket and object key of a (valid) reference.
func (s *service) location(ref string) (string, string) {
	var council, rest, found = strings.Cut(ref, "/")
	if id, err := uuid.Parse(council); found && err == nil {
		return councilBucket(id), rest
	}
	return s.bucket, ref
}
