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
		if n, err := fmt.Sscanf(v, "%d", &bufferSize); n != 1 || err != nil {
			log.Printf("[WARN] BUFFER_SIZE env var could not be parsed: %q", v)
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
	metaData, err := json.MarshalIndent(paste, "", "  ")
	if err != nil {
		return err
	}
	if fs.useS3 {
		key := paste.ID + ".json"
		input := &s3.PutObjectInput{
			Bucket:        aws.String(fs.s3Bucket),
			Key:           aws.String(key),
			Body:          bytes.NewReader(metaData),
			ContentLength: aws.Int64(int64(len(metaData))),
			ContentType:   aws.String("application/json"),
		}
		_, err := fs.s3Client.PutObject(context.Background(), input)
		if err != nil {
			log.Printf("[ERROR] S3 Store: failed to write metadata for %s: %v", paste.ID, err)
			return err
		}
		return nil
	}
	// Local FS
	metaPath := filepath.Join(fs.dataDir, paste.ID+".json")
	if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
		log.Printf("[ERROR] FS Store: failed to write metadata for %s: %v", paste.ID, err)
		return err
	}
	return nil
}

func (fs *FilesystemStore) Get(id string) (*models.Paste, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	var metaData []byte
	if fs.useS3 {
		key := id + ".json"
		input := &s3.GetObjectInput{
			Bucket: aws.String(fs.s3Bucket),
			Key:    aws.String(key),
		}
		out, err := fs.s3Client.GetObject(context.Background(), input)
		if err != nil {
			log.Printf("[ERROR] S3 Get: failed to read metadata for %s: %v", id, err)
			return nil, err
		}
		errClose := out.Body.Close()
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, out.Body); err != nil {
			log.Printf("[ERROR] S3 Get: failed to copy metadata for %s: %v", id, err)
			return nil, err
		}
		if errClose != nil {
			log.Printf("[WARN] S3 Get: error closing body for %s: %v", id, errClose)
		}
		metaData = buf.Bytes()
	} else {
		metaPath := filepath.Join(fs.dataDir, id+".json")
		var err error
		metaData, err = os.ReadFile(metaPath)
		if err != nil {
			log.Printf("[ERROR] FS Get: failed to read metadata for %s: %v", id, err)
			return nil, err
		}
	}
	var paste models.Paste
	if err := json.Unmarshal(metaData, &paste); err != nil {
		log.Printf("[ERROR] Get: failed to unmarshal metadata for %s: %v", id, err)
		return nil, err
	}
	return &paste, nil
}

func (fs *FilesystemStore) Delete(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.useS3 {
		// Remove content and metadata from S3
		keys := []string{id, id + ".json"}
		for _, key := range keys {
			input := &s3.DeleteObjectInput{
				Bucket: aws.String(fs.s3Bucket),
				Key:    aws.String(key),
			}
			_, err := fs.s3Client.DeleteObject(context.Background(), input)
			if err != nil {
				log.Printf("[ERROR] S3 Delete: failed to delete %s: %v", key, err)
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
		key := id + ".json"
		input := &s3.GetObjectInput{
			Bucket: aws.String(fs.s3Bucket),
			Key:    aws.String(key),
		}
		out, err := fs.s3Client.GetObject(context.Background(), input)
		if err != nil {
			log.Printf("[ERROR] S3 IncrementReadCount: failed to read metadata for %s: %v", id, err)
			return err
		}
		errClose := out.Body.Close()
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, out.Body); err != nil {
			log.Printf("[ERROR] S3 IncrementReadCount: failed to copy metadata for %s: %v", id, err)
			return err
		}
		if errClose != nil {
			log.Printf("[WARN] S3 IncrementReadCount: error closing body for %s: %v", id, errClose)
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
			log.Printf("[ERROR] S3 IncrementReadCount: failed to write metadata for %s: %v", id, err)
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
		errClose := out.Body.Close()
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, out.Body); err != nil {
			log.Printf("[ERROR] S3 GetContent: failed to copy content for %s: %v", id, err)
			return nil, err
		}
		if errClose != nil {
			log.Printf("[WARN] S3 GetContent: error closing body for %s: %v", id, errClose)
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
