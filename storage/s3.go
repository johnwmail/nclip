package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/johnwmail/nclip/models"
)

type S3Store struct {
	bucket string
	client *s3.Client
}

func NewS3Store(bucket string) (*S3Store, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg)
	return &S3Store{bucket: bucket, client: client}, nil
}

func (s *S3Store) Store(paste *models.Paste) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Store content (empty, as content is not in struct)
	contentKey := paste.ID
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(contentKey),
		Body:   bytes.NewReader([]byte{}),
	})
	if err != nil {
		return err
	}
	// Store metadata
	metaKey := paste.ID + ".json"
	metaData, err := json.MarshalIndent(paste, "", "  ")
	if err != nil {
		return err
	}
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(metaKey),
		Body:   bytes.NewReader(metaData),
	})
	return err
}

func (s *S3Store) Get(id string) (*models.Paste, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	metaKey := id + ".json"
	obj, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(metaKey),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = obj.Body.Close()
	}()
	metaData, err := io.ReadAll(obj.Body)
	if err != nil {
		return nil, err
	}
	var paste models.Paste
	if err := json.Unmarshal(metaData, &paste); err != nil {
		return nil, err
	}
	return &paste, nil
}

func (s *S3Store) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),
	})
	_, _ = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id + ".json"),
	})
	return nil
}

func (s *S3Store) IncrementReadCount(id string) error {
	paste, err := s.Get(id)
	if err != nil {
		return err
	}
	paste.ReadCount++
	return s.Store(paste)
}

func (s *S3Store) StoreContent(id string, content []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),
		Body:   bytes.NewReader(content),
	})
	return err
}

func (s *S3Store) GetContent(id string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	obj, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(id),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = obj.Body.Close()
	}()
	return io.ReadAll(obj.Body)
}

func (s *S3Store) Close() error {
	return nil
}
