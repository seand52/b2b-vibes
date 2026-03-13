package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const maxConcurrentUploads = 10

// s3API defines the S3 operations we use (for testing)
type s3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// Client wraps the AWS S3 client
type Client struct {
	s3     s3API
	bucket string
	region string
}

// Config holds S3 client configuration
type Config struct {
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

// UploadItem represents a single item to upload
type UploadItem struct {
	Key         string
	Data        []byte
	ContentType string
}

// UploadResult represents the result of an upload
type UploadResult struct {
	Key string
	URL string
	Err error
}

// ErrPartialUpload is returned when some uploads in a batch failed
var ErrPartialUpload = errors.New("some uploads failed")

// NewClient creates a new S3 client
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	return &Client{
		s3:     s3.NewFromConfig(awsCfg),
		bucket: cfg.Bucket,
		region: cfg.Region,
	}, nil
}

// Upload uploads data to S3 and returns the public URL
func (c *Client) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("uploading to S3: %w", err)
	}

	return c.buildURL(key), nil
}

// UploadBatch uploads multiple items concurrently.
// Returns all results and ErrPartialUpload if any uploads failed.
func (c *Client) UploadBatch(ctx context.Context, items []UploadItem) ([]UploadResult, error) {
	results := make([]UploadResult, len(items))
	sem := make(chan struct{}, maxConcurrentUploads)
	var wg sync.WaitGroup
	var failCount int
	var mu sync.Mutex

	for i, item := range items {
		wg.Add(1)
		go func(idx int, item UploadItem) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			url, err := c.Upload(ctx, item.Key, item.Data, item.ContentType)
			results[idx] = UploadResult{
				Key: item.Key,
				URL: url,
				Err: err,
			}

			if err != nil {
				mu.Lock()
				failCount++
				mu.Unlock()
			}
		}(i, item)
	}

	wg.Wait()

	if failCount > 0 {
		return results, ErrPartialUpload
	}
	return results, nil
}

// Delete removes an object from S3
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("deleting from S3: %w", err)
	}
	return nil
}

func (c *Client) buildURL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, c.region, key)
}
