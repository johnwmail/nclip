package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/johnwmail/nclip/models"
	"github.com/johnwmail/nclip/utils"
)

type S3Store struct {
	bucket string
	prefix string
	client *s3.Client
}

// NewS3Store creates a new S3Store instance
func NewS3Store(bucket, prefix string) (*S3Store, error) {
	if bucket == "" {
		return nil, fmt.Errorf("s3 bucket name must not be empty")
	}
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := s3.NewFromConfig(cfg)
	return &S3Store{bucket: bucket, prefix: prefix, client: client}, nil
}

func (s *S3Store) Store(paste *models.Paste) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Store metadata
	metaKey := applyS3Prefix(s.prefix, paste.ID+".json")
	metaData, err := json.MarshalIndent(paste, "", "  ")
	if err != nil {
		log.Printf("[ERROR] S3 Store: failed to marshal metadata for %s: %v", paste.ID, err)
		return err
	}
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(metaKey),
		Body:   bytes.NewReader(metaData),
	})
	if err != nil {
		log.Printf("[ERROR] S3 Store: failed to put metadata for %s: %v", paste.ID, err)
		if utils.IsDebugEnabled() {
			log.Printf("[DEBUG] S3 bucket: %s, prefix: %s, metaKey: %s", s.bucket, s.prefix, metaKey)
			for _, e := range os.Environ() {
				log.Printf("[ENV] %s", e)
			}
		}
		if awsErr, ok := err.(interface {
			ErrorCode() string
			ErrorMessage() string
		}); ok {
			log.Printf("[AWS ERROR] Code: %s, Message: %s", awsErr.ErrorCode(), awsErr.ErrorMessage())
		}
	}
	return err
}

func (s *S3Store) Get(id string) (*models.Paste, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	metaKey := applyS3Prefix(s.prefix, id+".json")
	obj, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(metaKey),
	})
	if err != nil {
		log.Printf("[ERROR] S3 Get: failed to get metadata for %s: %v", id, err)
		return nil, err
	}
	defer func() {
		if cerr := obj.Body.Close(); cerr != nil {
			log.Printf("[WARN] S3 Get: failed to close response body for %s: %v", id, cerr)
		}
	}()
	metaData, err := io.ReadAll(obj.Body)
	if err != nil {
		log.Printf("[ERROR] S3 Get: failed to read metadata body for %s: %v", id, err)
		return nil, err
	}
	var paste models.Paste
	if err := json.Unmarshal(metaData, &paste); err != nil {
		log.Printf("[ERROR] S3 Get: failed to unmarshal metadata for %s: %v", id, err)
		return nil, err
	}
	if paste.IsExpired() {
		log.Printf("[INFO] S3 Get: paste %s is expired", id)
		// Attempt to delete expired objects (best-effort)
		if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(applyS3Prefix(s.prefix, id)),
		}); err != nil {
			log.Printf("[WARN] S3 Get: failed to delete expired content for %s: %v", id, err)
		}
		if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(applyS3Prefix(s.prefix, id+".json")),
		}); err != nil {
			log.Printf("[WARN] S3 Get: failed to delete expired metadata for %s: %v", id, err)
		}
		return nil, fmt.Errorf("paste expired")
	}
	return &paste, nil
}

func (s *S3Store) Exists(id string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	metaKey := applyS3Prefix(s.prefix, id+".json")
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(metaKey),
	})
	if err != nil {
		// Check for AWS API errors
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			errorCode := apiErr.ErrorCode()
			// Object doesn't exist - this is what we want for available slugs
			if errorCode == "NoSuchKey" || errorCode == "NotFound" || errorCode == "404" {
				return false, nil
			}
		}
		// Also check for HTTP 404 in error message
		if strings.Contains(err.Error(), "StatusCode: 404") || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		// For any other error, log it but don't fail - treat as not exists to be safe
		// This handles cases where there might be temporary permission issues
		log.Printf("[WARN] S3 Exists: error checking %s, treating as not exists: %v", id, err)
		return false, nil
	}
	return true, nil
}

func (s *S3Store) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(applyS3Prefix(s.prefix, id)),
	})
	_, _ = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(applyS3Prefix(s.prefix, id+".json")),
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
		Key:    aws.String(applyS3Prefix(s.prefix, id)),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		log.Printf("[ERROR] S3 StoreContent: failed to put content for %s: %v", id, err)
		if utils.IsDebugEnabled() {
			log.Printf("[DEBUG] S3 bucket: %s, prefix: %s, key: %s", s.bucket, s.prefix, applyS3Prefix(s.prefix, id))
			for _, e := range os.Environ() {
				log.Printf("[ENV] %s", e)
			}
		}
		if awsErr, ok := err.(interface {
			ErrorCode() string
			ErrorMessage() string
		}); ok {
			log.Printf("[AWS ERROR] Code: %s, Message: %s", awsErr.ErrorCode(), awsErr.ErrorMessage())
		}
	}
	return err
}

func (s *S3Store) GetContent(id string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	obj, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(applyS3Prefix(s.prefix, id)),
	})
	if err != nil {
		log.Printf("[ERROR] S3 GetContent: failed to get content for %s: %v", id, err)
		return nil, err
	}
	defer func() {
		if cerr := obj.Body.Close(); cerr != nil {
			log.Printf("[WARN] S3 GetContent: failed to close response body for %s: %v", id, cerr)
		}
	}()
	data, err := io.ReadAll(obj.Body)
	if err != nil {
		log.Printf("[ERROR] S3 GetContent: failed to read content body for %s: %v", id, err)
		return nil, err
	}
	return data, nil
}

// StatContent checks if the object exists in S3 and returns its size.
func (s *S3Store) StatContent(id string) (bool, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	key := applyS3Prefix(s.prefix, id)
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			code := apiErr.ErrorCode()
			if code == "NotFound" || code == "NoSuchKey" || code == "404" {
				return false, 0, nil
			}
		}
		// Also check textual message for 404
		if strings.Contains(err.Error(), "StatusCode: 404") || strings.Contains(err.Error(), "NotFound") {
			return false, 0, nil
		}
		log.Printf("[WARN] S3 StatContent: error checking %s: %v", id, err)
		return false, 0, err
	}
	if head.ContentLength == nil {
		return true, 0, nil
	}
	return true, *head.ContentLength, nil
}

// GetContentPrefix retrieves up to n bytes from an S3 object using Range header.
func (s *S3Store) GetContentPrefix(id string, n int64) ([]byte, error) {
	if n <= 0 {
		return []byte{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	key := applyS3Prefix(s.prefix, id)
	// Range header is inclusive: bytes=0-(n-1)
	rangeHeader := fmt.Sprintf("bytes=0-%d", n-1)
	obj, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Range:  aws.String(rangeHeader),
	})
	if err != nil {
		log.Printf("[ERROR] S3 GetContentPrefix: failed to get object %s: %v", id, err)
		return nil, err
	}
	defer func() {
		if cerr := obj.Body.Close(); cerr != nil {
			log.Printf("[WARN] S3 GetContentPrefix: failed to close response body for %s: %v", id, cerr)
		}
	}()
	data, err := io.ReadAll(obj.Body)
	if err != nil {
		log.Printf("[ERROR] S3 GetContentPrefix: failed to read body for %s: %v", id, err)
		return nil, err
	}
	return data, nil
}

func (s *S3Store) Close() error {
	return nil
}
