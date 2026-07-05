package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jmoiron/sqlx"
)

type StorageService interface {
	Upload(ctx context.Context, path string, reader io.Reader) (string, error)
	Delete(ctx context.Context, path string) error
	GetURL(path string) string
}

// ==================== S3 STORAGE ====================

type S3Storage struct {
	client *s3.Client
	bucket string
	region string
	prefix string // optional path prefix, e.g. "bus-images/"
}

func NewS3Storage(region, bucket, prefix string) (*S3Storage, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	return &S3Storage{
		client: client,
		bucket: bucket,
		region: region,
		prefix: prefix,
	}, nil
}

func (s *S3Storage) Upload(ctx context.Context, path string, reader io.Reader) (string, error) {
	key := s.prefix + path
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}
	return s.GetURL(path), nil
}

func (s *S3Storage) Delete(ctx context.Context, path string) error {
	key := s.prefix + path
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *S3Storage) GetURL(path string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s%s", s.bucket, s.region, s.prefix, path)
}

// ==================== LOCAL STORAGE (fallback for dev) ====================

type LocalStorage struct {
	basePath string
	baseURL  string // e.g. "http://localhost:8080/uploads"
}

func NewLocalStorage(basePath, baseURL string) *LocalStorage {
	os.MkdirAll(basePath, 0755)
	return &LocalStorage{basePath: basePath, baseURL: baseURL}
}

func (s *LocalStorage) Upload(ctx context.Context, path string, reader io.Reader) (string, error) {
	fullPath := filepath.Join(s.basePath, path)
	os.MkdirAll(filepath.Dir(fullPath), 0755)

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return s.GetURL(path), nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	return os.Remove(filepath.Join(s.basePath, path))
}

func (s *LocalStorage) GetURL(path string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, path)
}

// ==================== PG STORAGE (BYTEA in PostgreSQL) ====================

type PgStorage struct {
	db      *sqlx.DB
	baseURL string
}

func NewPgStorage(db *sqlx.DB, baseURL string) *PgStorage {
	return &PgStorage{db: db, baseURL: baseURL}
}

func (s *PgStorage) Upload(ctx context.Context, path string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	contentType := "image/jpeg"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	// Upsert: insert or update if file_name already exists (preserve original created_at)
	_, err = s.db.Exec(`
		INSERT INTO bus_images (file_name, image_data, content_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (file_name) DO UPDATE SET
			image_data = $2, content_type = $3`,
		path, data, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to store image in database: %w", err)
	}

	return s.GetURL(path), nil
}

func (s *PgStorage) Delete(ctx context.Context, path string) error {
	_, err := s.db.Exec(`DELETE FROM bus_images WHERE file_name = $1`, path)
	return err
}

func (s *PgStorage) GetURL(path string) string {
	return fmt.Sprintf("%s/api/images/%s", s.baseURL, path)
}

// ==================== HELPER ====================

func GenerateFilePath(extension string) string {
	return fmt.Sprintf("%d%s", time.Now().UnixNano(), extension)
}
