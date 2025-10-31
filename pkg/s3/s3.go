package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	client    *minio.Client
	bucket    string
	directory string
}

func New(endpoint string, region string, accessKey string, secretKey string, useSSL bool, bucket string, directory string) (*S3, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Region: region,
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{Region: region, ObjectLocking: false})
		if err != nil {
			return nil, err
		}
	}

	return &S3{
		client:    client,
		bucket:    bucket,
		directory: directory,
	}, nil
}

func (s *S3) Objects(ctx context.Context) ([]string, error) {
	var result []string

	objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    s.directory,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}

		name := object.Key
		if len(s.directory) > 0 {
			if len(name) > len(s.directory)+1 {
				name = name[len(s.directory)+1:]
			} else {
				continue
			}
		}

		result = append(result, name)
	}

	return result, nil
}

func (s *S3) Upload(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, s.object(filename), reader, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *S3) Download(ctx context.Context, filename string) (*minio.Object, error) {
	return s.client.GetObject(ctx, s.bucket, s.object(filename), minio.GetObjectOptions{})
}

func (s *S3) Delete(ctx context.Context, filename string) error {
	return s.client.RemoveObject(ctx, s.bucket, s.object(filename), minio.RemoveObjectOptions{})
}

func (s *S3) Exists(ctx context.Context, filename string) bool {
	_, err := s.client.StatObject(ctx, s.bucket, s.object(filename), minio.StatObjectOptions{})
	return err == nil
}

func (s *S3) Rename(ctx context.Context, oldFilename string, newFilename string) error {
	src := minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: s.object(oldFilename),
	}
	dest := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: s.object(newFilename),
	}
	_, err := s.client.CopyObject(ctx, dest, src)
	if err != nil {
		return err
	}
	return s.Delete(ctx, oldFilename)
}

func (s *S3) object(filename string) string {
	if len(s.directory) == 0 {
		return filename
	}
	return fmt.Sprintf("%s/%s", s.directory, filename)
}
