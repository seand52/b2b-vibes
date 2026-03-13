package s3

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockS3API implements s3API for testing
type mockS3API struct {
	putObjectFunc    func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	deleteObjectFunc func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

func (m *mockS3API) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3API) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.deleteObjectFunc != nil {
		return m.deleteObjectFunc(ctx, params, optFns...)
	}
	return &s3.DeleteObjectOutput{}, nil
}

func TestUpload(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		data        []byte
		contentType string
		mockErr     error
		wantURL     string
		wantErr     bool
	}{
		{
			name:        "successful upload",
			key:         "products/123/image.jpg",
			data:        []byte("image data"),
			contentType: "image/jpeg",
			wantURL:     "https://test-bucket.s3.eu-west-1.amazonaws.com/products/123/image.jpg",
		},
		{
			name:        "upload fails",
			key:         "products/123/image.jpg",
			data:        []byte("image data"),
			contentType: "image/jpeg",
			mockErr:     errors.New("access denied"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockS3API{
				putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					assert.Equal(t, "test-bucket", *params.Bucket)
					assert.Equal(t, tt.key, *params.Key)
					assert.Equal(t, tt.contentType, *params.ContentType)
					return &s3.PutObjectOutput{}, tt.mockErr
				},
			}

			client := &Client{
				s3:     mock,
				bucket: "test-bucket",
				region: "eu-west-1",
			}

			url, err := client.Upload(context.Background(), tt.key, tt.data, tt.contentType)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, url)
		})
	}
}

func TestUploadBatch(t *testing.T) {
	t.Run("all uploads succeed", func(t *testing.T) {
		uploadCount := 0
		mock := &mockS3API{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				uploadCount++
				return &s3.PutObjectOutput{}, nil
			},
		}

		client := &Client{
			s3:     mock,
			bucket: "test-bucket",
			region: "eu-west-1",
		}

		items := []UploadItem{
			{Key: "img1.jpg", Data: []byte("data1"), ContentType: "image/jpeg"},
			{Key: "img2.jpg", Data: []byte("data2"), ContentType: "image/jpeg"},
			{Key: "img3.jpg", Data: []byte("data3"), ContentType: "image/jpeg"},
		}

		results, err := client.UploadBatch(context.Background(), items)

		require.NoError(t, err)
		assert.Len(t, results, 3)
		assert.Equal(t, 3, uploadCount)

		for _, r := range results {
			assert.NoError(t, r.Err)
			assert.NotEmpty(t, r.URL)
		}
	})

	t.Run("some uploads fail returns ErrPartialUpload", func(t *testing.T) {
		mock := &mockS3API{
			putObjectFunc: func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				// Fail uploads for img2.jpg
				if *params.Key == "img2.jpg" {
					return nil, errors.New("upload failed")
				}
				return &s3.PutObjectOutput{}, nil
			},
		}

		client := &Client{
			s3:     mock,
			bucket: "test-bucket",
			region: "eu-west-1",
		}

		items := []UploadItem{
			{Key: "img1.jpg", Data: []byte("data1"), ContentType: "image/jpeg"},
			{Key: "img2.jpg", Data: []byte("data2"), ContentType: "image/jpeg"},
			{Key: "img3.jpg", Data: []byte("data3"), ContentType: "image/jpeg"},
		}

		results, err := client.UploadBatch(context.Background(), items)

		assert.ErrorIs(t, err, ErrPartialUpload)
		assert.Len(t, results, 3)

		// Check individual results
		assert.NoError(t, results[0].Err)
		assert.Error(t, results[1].Err)
		assert.NoError(t, results[2].Err)
	})

	t.Run("empty batch", func(t *testing.T) {
		client := &Client{
			s3:     &mockS3API{},
			bucket: "test-bucket",
			region: "eu-west-1",
		}

		results, err := client.UploadBatch(context.Background(), []UploadItem{})

		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		mockErr error
		wantErr bool
	}{
		{
			name: "successful delete",
			key:  "products/123/image.jpg",
		},
		{
			name:    "delete fails",
			key:     "products/123/image.jpg",
			mockErr: errors.New("not found"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockS3API{
				deleteObjectFunc: func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
					assert.Equal(t, "test-bucket", *params.Bucket)
					assert.Equal(t, tt.key, *params.Key)
					return &s3.DeleteObjectOutput{}, tt.mockErr
				},
			}

			client := &Client{
				s3:     mock,
				bucket: "test-bucket",
				region: "eu-west-1",
			}

			err := client.Delete(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
