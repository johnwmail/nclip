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
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/johnwmail/nclip/models"
	"github.com/johnwmail/nclip/utils"
)

// Store saves the paste metadata (JSON) to local filesystem or S3
func (fs *FilesystemStore) Store(paste *models.Paste) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	metaData, err := json.MarshalIndent(paste, "", "  ")
	if err != nil {
		log.Printf("[ERROR] FS Store: failed to marshal metadata for %s: %v", paste.ID, err)
		return err
	}
	if fs.useS3 {
		key := fs.s3Key(paste.ID + ".json")
		input := &s3.PutObjectInput{
			Bucket:        aws.String(fs.s3Bucket),
			Key:           aws.String(key),
			Body:          bytes.NewReader(metaData),
			ContentLength: aws.Int64(int64(len(metaData))),
			ContentType:   aws.String("application/json"),
		}
		_, err := fs.s3Client.PutObject(context.Background(), input)
		if err != nil {
			logAwsError(fmt.Sprintf("S3 Store metadata for %s", paste.ID), err)
			return err
		}
		return nil
	}
	// Local FS
	// Ensure data directory exists (may have been removed after startup by external cleanup)
	if err := os.MkdirAll(fs.dataDir, 0o755); err != nil {
		log.Printf("[ERROR] FS Store: failed to create data directory %s: %v", fs.dataDir, err)
		return err
	}
	metaPath := filepath.Join(fs.dataDir, paste.ID+".json")
	if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
		log.Printf("[ERROR] FS Store: failed to write metadata for %s: %v", paste.ID, err)
		return err
	}
	return nil
}

type FilesystemStore struct {
	dataDir    string
	s3Bucket   string
	s3Prefix   string
	useS3      bool
	bufferSize int
	mu         sync.Mutex
	s3Client   *s3.Client
}

func NewFilesystemStore() (*FilesystemStore, error) {
	dataDir := os.Getenv("NCLIP_DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// Create the data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Printf("[ERROR] Failed to create data directory %s: %v", dataDir, err)
		return nil, fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	} else {
		log.Printf("[INFO] Created data directory: %s", dataDir)
	}

	s3Bucket := os.Getenv("NCLIP_S3_BUCKET")
	// TODO: Implement initialization logic if needed
	return &FilesystemStore{
		dataDir:    dataDir,
		s3Bucket:   s3Bucket,
		s3Prefix:   "",
		useS3:      s3Bucket != "",
		bufferSize: 4096,
		s3Client:   nil, // Should be initialized if useS3 is true
	}, nil
}

func (fs *FilesystemStore) Get(id string) (*models.Paste, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	var metaData []byte
	if fs.useS3 {
		key := fs.s3Key(id + ".json")
		input := &s3.GetObjectInput{
			Bucket: aws.String(fs.s3Bucket),
			Key:    aws.String(key),
		}
		out, err := fs.s3Client.GetObject(context.Background(), input)
		if err != nil {
			var apiErr smithy.APIError
			if !errors.As(err, &apiErr) || apiErr.ErrorCode() != "NoSuchKey" {
				logAwsError(fmt.Sprintf("S3 Get metadata for %s", id), err)
			}
			return nil, err
		}
		defer func() {
			if errClose := out.Body.Close(); errClose != nil {
				log.Printf("[WARN] S3 Get: error closing body for %s: %v", id, errClose)
			}
		}()
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, out.Body); err != nil {
			log.Printf("[ERROR] S3 Get: failed to copy metadata for %s: %v", id, err)
			return nil, err
		}
		metaData = buf.Bytes()
	} else {
		metaPath := filepath.Join(fs.dataDir, id+".json")
		var err error
		metaData, err = os.ReadFile(metaPath)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("[ERROR] FS Get: failed to read metadata for %s: %v", id, err)
			}
			return nil, err
		}
	}
	var paste models.Paste
	if err := json.Unmarshal(metaData, &paste); err != nil {
		log.Printf("[ERROR] Get: failed to unmarshal metadata for %s: %v", id, err)
		return nil, err
	}
	if paste.IsExpired() {
		log.Printf("[INFO] FS Get: paste %s is expired", id)
		return nil, os.ErrNotExist
	}
	return &paste, nil
}

func (fs *FilesystemStore) Exists(id string) (bool, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.useS3 {
		key := fs.s3Key(id + ".json")
		input := &s3.HeadObjectInput{
			Bucket: aws.String(fs.s3Bucket),
			Key:    aws.String(key),
		}
		_, err := fs.s3Client.HeadObject(context.Background(), input)
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NoSuchKey" {
				return false, nil
			}
			logAwsError(fmt.Sprintf("S3 Exists check for %s", id), err)
			return false, err
		}
		return true, nil
	}
	// Local FS
	metaPath := filepath.Join(fs.dataDir, id+".json")
	_, err := os.Stat(metaPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		log.Printf("[ERROR] FS Exists: failed to stat metadata for %s: %v", id, err)
		return false, err
	}
	return true, nil
}

func (fs *FilesystemStore) Delete(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.useS3 {
		// Remove content and metadata from S3
		keys := []string{fs.s3Key(id), fs.s3Key(id + ".json")}
		for _, key := range keys {
			input := &s3.DeleteObjectInput{
				Bucket: aws.String(fs.s3Bucket),
				Key:    aws.String(key),
			}
			_, err := fs.s3Client.DeleteObject(context.Background(), input)
			if err != nil {
				logAwsError(fmt.Sprintf("S3 Delete object %s", key), err)
			}
		}
		return nil
	}
	// Local FS
	contentPath := filepath.Join(fs.dataDir, id)
	metaPath := filepath.Join(fs.dataDir, id+".json")
	_ = os.Remove(contentPath)
	_ = os.Remove(metaPath)
	return nil
}

func (fs *FilesystemStore) IncrementReadCount(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	var metaData []byte
	if fs.useS3 {
		key := fs.s3Key(id + ".json")
		input := &s3.GetObjectInput{
			Bucket: aws.String(fs.s3Bucket),
			Key:    aws.String(key),
		}
		out, err := fs.s3Client.GetObject(context.Background(), input)
		if err != nil {
			logAwsError(fmt.Sprintf("S3 IncrementReadCount read metadata for %s", id), err)
			return err
		}
		defer func() {
			if errClose := out.Body.Close(); errClose != nil {
				log.Printf("[WARN] S3 IncrementReadCount: error closing body for %s: %v", id, errClose)
			}
		}()
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, out.Body); err != nil {
			log.Printf("[ERROR] S3 IncrementReadCount: failed to copy metadata for %s: %v", id, err)
			return err
		}
		metaData = buf.Bytes()
		var paste models.Paste
		if err := json.Unmarshal(metaData, &paste); err != nil {
			log.Printf("[ERROR] S3 IncrementReadCount: failed to unmarshal metadata for %s: %v", id, err)
			return err
		}
		paste.ReadCount++
		newMeta, err := json.MarshalIndent(&paste, "", "  ")
		if err != nil {
			return err
		}
		putInput := &s3.PutObjectInput{
			Bucket:        aws.String(fs.s3Bucket),
			Key:           aws.String(key),
			Body:          bytes.NewReader(newMeta),
			ContentLength: aws.Int64(int64(len(newMeta))),
			ContentType:   aws.String("application/json"),
		}
		_, err = fs.s3Client.PutObject(context.Background(), putInput)
		if err != nil {
			logAwsError(fmt.Sprintf("S3 IncrementReadCount write metadata for %s", id), err)
			return err
		}
		return nil
	} else {
		metaPath := filepath.Join(fs.dataDir, id+".json")
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			log.Printf("[ERROR] FS IncrementReadCount: failed to read metadata for %s: %v", id, err)
			return err
		}
		var paste models.Paste
		if err := json.Unmarshal(metaData, &paste); err != nil {
			log.Printf("[ERROR] FS IncrementReadCount: failed to unmarshal metadata for %s: %v", id, err)
			return err
		}
		paste.ReadCount++
		newMeta, err := json.MarshalIndent(&paste, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(metaPath, newMeta, 0o644); err != nil {
			log.Printf("[ERROR] FS IncrementReadCount: failed to write metadata for %s: %v", id, err)
			return err
		}
		return nil
	}
}

func (fs *FilesystemStore) StoreContent(id string, content []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if utils.IsDebugEnabled() {
		log.Printf("[DEBUG] StoreContent: id=%s, content_len=%d, first_bytes=%q", id, len(content), string(content[:min(32, len(content))]))
	}
	if fs.useS3 {
		// S3 upload
		key := fs.s3Key(id)
		input := &s3.PutObjectInput{
			Bucket:        aws.String(fs.s3Bucket),
			Key:           aws.String(key),
			Body:          bytes.NewReader(content),
			ContentLength: aws.Int64(int64(len(content))),
			ContentType:   aws.String("application/octet-stream"),
		}
		_, err := fs.s3Client.PutObject(context.Background(), input)
		if err != nil {
			logAwsError(fmt.Sprintf("S3 StoreContent write content for %s", id), err)
			return err
		}
		return nil
	}
	// Local FS
	// Ensure data directory exists before attempting to write content. This avoids failures
	// when the directory was removed after startup (e.g., CI cleanup hooks).
	if err := os.MkdirAll(fs.dataDir, 0o755); err != nil {
		log.Printf("[ERROR] FS StoreContent: failed to create data directory %s: %v", fs.dataDir, err)
		return err
	}
	contentPath := filepath.Join(fs.dataDir, id)
	if err := os.WriteFile(contentPath, content, 0o644); err != nil {
		log.Printf("[ERROR] FS StoreContent: failed to write content for %s: %v", id, err)
		return err
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (fs *FilesystemStore) GetContent(id string) ([]byte, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.useS3 {
		key := fs.s3Key(id)
		input := &s3.GetObjectInput{
			Bucket: aws.String(fs.s3Bucket),
			Key:    aws.String(key),
		}
		out, err := fs.s3Client.GetObject(context.Background(), input)
		if err != nil {
			logAwsError(fmt.Sprintf("S3 GetContent read content for %s", id), err)
			return nil, err
		}
		defer func() {
			if errClose := out.Body.Close(); errClose != nil {
				log.Printf("[WARN] S3 GetContent: error closing body for %s: %v", id, errClose)
			}
		}()
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, out.Body); err != nil {
			log.Printf("[ERROR] S3 GetContent: failed to copy content for %s: %v", id, err)
			return nil, err
		}
		return buf.Bytes(), nil
	}
	contentPath := filepath.Join(fs.dataDir, id)
	data, err := os.ReadFile(contentPath)
	if err != nil {
		log.Printf("[ERROR] FS GetContent: failed to read content for %s: %v", id, err)
		return nil, err
	}
	return data, nil
}

func (fs *FilesystemStore) Close() error {
	return nil
}

func (fs *FilesystemStore) s3Key(name string) string {
	return applyS3Prefix(fs.s3Prefix, name)
}

func logAwsError(ctx string, err error) {
	if err == nil {
		return
	}
	log.Printf("[ERROR] %s: %v", ctx, err)

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		log.Printf("[ERROR] %s: code=%s message=%s fault=%s", ctx, apiErr.ErrorCode(), apiErr.ErrorMessage(), apiErr.ErrorFault())
	}

	var respErr *smithyhttp.ResponseError
	if errors.As(err, &respErr) {
		if resp := respErr.HTTPResponse(); resp != nil {
			log.Printf("[ERROR] %s: status=%s request-id=%s extended-request-id=%s", ctx, resp.Status, resp.Header.Get("x-amz-request-id"), resp.Header.Get("x-amz-id-2"))
			if resp.Body != nil {
				body, readErr := io.ReadAll(resp.Body)
				if readErr == nil {
					trimmed := strings.TrimSpace(string(body))
					if trimmed != "" {
						const maxErrorBodyLog = 4096
						if len(trimmed) > maxErrorBodyLog {
							trimmed = trimmed[:maxErrorBodyLog] + "... (truncated)"
						}
						log.Printf("[ERROR] %s: body=%s", ctx, trimmed)
					}
				} else {
					log.Printf("[WARN] %s: failed to read error body: %v", ctx, readErr)
				}
				if closeErr := resp.Body.Close(); closeErr != nil {
					log.Printf("[WARN] %s: failed to close error body: %v", ctx, closeErr)
				}
			}
		}
	}
}
