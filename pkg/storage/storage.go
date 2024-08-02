package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/yezzey-gp/aws-sdk-go/aws"
	"github.com/yezzey-gp/aws-sdk-go/service/s3"
	"github.com/yezzey-gp/aws-sdk-go/service/s3/s3manager"
	"github.com/yezzey-gp/yproxy/config"
	"github.com/yezzey-gp/yproxy/pkg/ylogger"
)

type StorageReader interface {
	CatFileFromStorage(name string, offset int64) (io.ReadCloser, error)
	ListPath(name string) ([]*S3ObjectMeta, error)
}

type StorageWriter interface {
	PutFileToDest(name string, r io.Reader) error
	PatchFile(name string, r io.ReadSeeker, startOffste int64) error
}

type StorageLister interface {
	ListPath(prefix string) ([]*S3ObjectMeta, error)
}

type StorageMover interface {
	MoveObject(from string, to string) error
}

type StorageInteractor interface {
	StorageReader
	StorageWriter
	StorageLister
	StorageMover
}

type S3StorageInteractor struct {
	StorageInteractor

	pool SessionPool

	cnf *config.Storage
}

func NewStorage(cnf *config.Storage) StorageInteractor {
	return &S3StorageInteractor{
		pool: NewSessionPool(cnf),
		cnf:  cnf,
	}
}

func (s *S3StorageInteractor) CatFileFromStorage(name string, offset int64) (io.ReadCloser, error) {
	// XXX: fix this
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil, err
	}

	objectPath := path.Join(s.cnf.StoragePrefix, name)
	input := &s3.GetObjectInput{
		Bucket: &s.cnf.StorageBucket,
		Key:    aws.String(objectPath),
		Range:  aws.String(fmt.Sprintf("bytes=%d-", offset)),
	}

	ylogger.Zero.Debug().Str("key", objectPath).Int64("offset", offset).Str("bucket",
		s.cnf.StorageBucket).Msg("requesting external storage")

	object, err := sess.GetObject(input)
	return object.Body, err
}

func (s *S3StorageInteractor) PutFileToDest(name string, r io.Reader) error {
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil
	}

	objectPath := path.Join(s.cnf.StoragePrefix, name)

	up := s3manager.NewUploaderWithClient(sess, func(uploader *s3manager.Uploader) {
		uploader.PartSize = int64(1 << 24)
		uploader.Concurrency = 1
	})

	_, err = up.Upload(
		&s3manager.UploadInput{
			Bucket:       aws.String(s.cnf.StorageBucket),
			Key:          aws.String(objectPath),
			Body:         r,
			StorageClass: aws.String("STANDARD"),
		},
	)

	return err
}

func (s *S3StorageInteractor) PatchFile(name string, r io.ReadSeeker, startOffste int64) error {
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil
	}

	objectPath := path.Join(s.cnf.StoragePrefix, name)

	input := &s3.PatchObjectInput{
		Bucket:       &s.cnf.StorageBucket,
		Key:          aws.String(objectPath),
		Body:         r,
		ContentRange: aws.String(fmt.Sprintf("bytes %d-18446744073709551615", startOffste)),
	}

	_, err = sess.PatchObject(input)

	ylogger.Zero.Debug().Str("key", objectPath).Str("bucket",
		s.cnf.StorageBucket).Msg("modifying file in external storage")

	return err
}

type S3ObjectMeta struct {
	Path         string
	Size         int64
	LastModified time.Time
}

func (s *S3StorageInteractor) ListPath(prefix string) ([]*S3ObjectMeta, error) {
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return nil, err
	}

	var continuationToken *string
	prefix = path.Join(s.cnf.StoragePrefix, prefix)
	metas := make([]*S3ObjectMeta, 0)

	for {
		input := &s3.ListObjectsV2Input{
			Bucket:            &s.cnf.StorageBucket,
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		}

		out, err := sess.ListObjectsV2(input)
		if err != nil {
			fmt.Printf("list error: %v\n", err)
		}

		for _, obj := range out.Contents {
			metas = append(metas, &S3ObjectMeta{
				Path:         *obj.Key,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})
		}

		if !*out.IsTruncated {
			break
		}

		continuationToken = out.NextContinuationToken
	}
	return metas, nil
}

func (s *S3StorageInteractor) MoveObject(from string, to string) error {
	sess, err := s.pool.GetSession(context.TODO())
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to acquire s3 session")
		return err
	}

	fromPath := path.Join(s.cnf.StoragePrefix, from)
	toPath := path.Join(s.cnf.StoragePrefix, to)

	input := s3.CopyObjectInput{
		Bucket:     &s.cnf.StorageBucket,
		CopySource: aws.String(fromPath),
		Key:        aws.String(toPath),
	}

	_, err = sess.CopyObject(&input)
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to copy object")
		return err
	}

	input2 := s3.DeleteObjectInput{
		Bucket: &s.cnf.StorageBucket,
		Key:    aws.String(fromPath),
	}

	_, err = sess.DeleteObject(&input2)
	if err != nil {
		ylogger.Zero.Err(err).Msg("failed to delete old object")
	}
	return err
}
