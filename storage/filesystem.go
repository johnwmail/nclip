package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/johnwmail/nclip/models"
)

type FilesystemStore struct {
	dataDir    string
	s3Bucket   string
	useS3      bool
	bufferSize int
	mu         sync.Mutex
	s3Client   *s3.Client
}

func NewFilesystemStore() (*FilesystemStore, error) {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/tmp"
	}
	s3Bucket := os.Getenv("S3_BUCKET")
	useS3 := s3Bucket != ""
	bufferSize := 50 * 1024 * 1024 // 50MB default
	if v := os.Getenv("BUFFER_SIZE"); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &bufferSize); n == 1 && err == nil {
			// parsed
		}
	}
	var s3Client *s3.Client
	if useS3 {
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			log.Printf("[ERROR] S3 config: %v", err)
			return nil, err
		}
		s3Client = s3.NewFromConfig(cfg)
	}
	if !useS3 {
		if err := os.MkdirAll(dataDir, 0o755); err != nil {
			return nil, err
		}
	}
	return &FilesystemStore{
		dataDir:    dataDir,
		s3Bucket:   s3Bucket,
		useS3:      useS3,
		bufferSize: bufferSize,
		s3Client:   s3Client,
	}, nil
}

func (fs *FilesystemStore) Store(paste *models.Paste) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	// Write metadata
	metaPath := filepath.Join(fs.dataDir, paste.ID+".json")
	metaData, err := json.MarshalIndent(paste, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
		log.Printf("[ERROR] FS Store: failed to write metadata for %s: %v", paste.ID, err)
		return err
	}
	return nil
}

func (fs *FilesystemStore) Get(id string) (*models.Paste, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	metaPath := filepath.Join(fs.dataDir, id+".json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		log.Printf("[ERROR] FS Get: failed to read metadata for %s: %v", id, err)
		return nil, err
	}
	var paste models.Paste
	if err := json.Unmarshal(metaData, &paste); err != nil {
		log.Printf("[ERROR] FS Get: failed to unmarshal metadata for %s: %v", id, err)
		return nil, err
	}
	return &paste, nil
}

func (fs *FilesystemStore) Delete(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	contentPath := filepath.Join(fs.dataDir, id)
	metaPath := filepath.Join(fs.dataDir, id+".json")
	_ = os.Remove(contentPath)
	_ = os.Remove(metaPath)
	return nil
}

func (fs *FilesystemStore) IncrementReadCount(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
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

func (fs *FilesystemStore) StoreContent(id string, content []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	log.Printf("[DEBUG] StoreContent: id=%s, content_len=%d, first_bytes=%q", id, len(content), string(content[:min(32, len(content))]))
	if fs.useS3 {
		// S3 upload
		key := id
		input := &s3.PutObjectInput{
			Bucket:        aws.String(fs.s3Bucket),
			Key:           aws.String(key),
			Body:          bytes.NewReader(content),
			ContentLength: aws.Int64(int64(len(content))),
			ContentType:   aws.String("application/octet-stream"),
		}
		_, err := fs.s3Client.PutObject(context.Background(), input)
		if err != nil {
			log.Printf("[ERROR] S3 StoreContent: failed to write content for %s: %v", id, err)
			return err
		}
		return nil
	}
	// Local FS
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
		key := id
		input := &s3.GetObjectInput{
			Bucket: aws.String(fs.s3Bucket),
			Key:    aws.String(key),
		}
		out, err := fs.s3Client.GetObject(context.Background(), input)
		if err != nil {
			log.Printf("[ERROR] S3 GetContent: failed to read content for %s: %v", id, err)
			return nil, err
		}
		defer out.Body.Close()
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
