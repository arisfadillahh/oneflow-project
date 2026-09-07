package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"oneflow/app-backend/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	client       *minio.Client
	publicClient *minio.Client
	bucket       string
	publicBase   string
}

func New(ctx context.Context, cfg config.Config) (*Client, error) {
	endpoint, _, err := normalizeEndpoint(cfg.ObjectStorageEndpoint)
	if err != nil {
		return nil, err
	}
	publicEndpoint, publicBase, err := normalizeEndpoint(cfg.ObjectStoragePublicBaseURL)
	if err != nil {
		return nil, err
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.ObjectStorageAccessKey, cfg.ObjectStorageSecretKey, ""),
		Secure: cfg.ObjectStorageUseSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, err
	}
	publicClient, err := minio.New(publicEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.ObjectStorageAccessKey, cfg.ObjectStorageSecretKey, ""),
		Secure: strings.HasPrefix(publicBase, "https://"),
		Region: "us-east-1",
	})
	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(ctx, cfg.ObjectStorageBucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.ObjectStorageBucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &Client{
		client:       client,
		publicClient: publicClient,
		bucket:       cfg.ObjectStorageBucket,
		publicBase:   strings.TrimRight(publicBase, "/"),
	}, nil
}

func (c *Client) UploadTextFile(ctx context.Context, objectKey string, content []byte, contentType string) (string, error) {
	return c.UploadFile(ctx, objectKey, content, contentType)
}

func (c *Client) UploadFile(ctx context.Context, objectKey string, content []byte, contentType string) (string, error) {
	_, err := c.client.PutObject(ctx, c.bucket, objectKey, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s", c.publicBase, c.bucket, objectKey), nil
}

func (c *Client) Download(ctx context.Context, objectKey string) ([]byte, error) {
	object, err := c.client.GetObject(ctx, c.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()
	return io.ReadAll(object)
}

func (c *Client) PresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = time.Hour
	}
	presignClient := c.publicClient
	if presignClient == nil {
		presignClient = c.client
	}
	u, err := presignClient.PresignedGetObject(ctx, c.bucket, objectKey, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *Client) Delete(ctx context.Context, objectKey string) error {
	return c.client.RemoveObject(ctx, c.bucket, objectKey, minio.RemoveObjectOptions{})
}

func normalizeEndpoint(raw string) (string, string, error) {
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", err
	}
	return parsed.Host, parsed.Scheme + "://" + parsed.Host, nil
}
