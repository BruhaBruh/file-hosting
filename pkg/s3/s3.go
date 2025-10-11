package s3

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	client *minio.Client
	bucket string
}

func New(endpoint string, region string, accessKey string, secretKey string, useSSL bool, bucket string) (*S3, error) {
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
		client: client,
		bucket: bucket,
	}, nil
}

func (s *S3) Upload(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, filename, reader, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *S3) Download(ctx context.Context, filename string) (*minio.Object, error) {
	return s.client.GetObject(ctx, s.bucket, filename, minio.GetObjectOptions{})
}

func (s *S3) Delete(ctx context.Context, filename string) error {
	return s.client.RemoveObject(ctx, s.bucket, filename, minio.RemoveObjectOptions{})
}

func (s *S3) Exists(ctx context.Context, filename string) bool {
	_, err := s.client.StatObject(ctx, s.bucket, filename, minio.StatObjectOptions{})
	return err == nil
}

func (s *S3) Rename(ctx context.Context, oldFilename string, newFilename string) error {
	src := minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: oldFilename,
	}
	dest := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: newFilename,
	}
	_, err := s.client.CopyObject(ctx, dest, src)
	if err != nil {
		return err
	}
	return s.Delete(ctx, oldFilename)
}
