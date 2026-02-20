package installer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	client     *minio.Client
	bucketName string
}

func NewMinioClient() (*MinioClient, error) {
	host := getEnv("MINIO_HOST", "127.0.0.1")
	port := getEnv("MINIO_PORT", "9000")
	accessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	secretKey := getEnv("MINIO_SECRET_KEY", "minioadmin")
	bucket := getEnv("MINIO_BUCKET", "reprobuild")

	endpoint := fmt.Sprintf("%s:%s", host, port)

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	return &MinioClient{
		client:     minioClient,
		bucketName: bucket,
	}, nil
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func (m *MinioClient) FileExists(ctx context.Context, hash string) (bool, error) {
	objectName := hash + ".zip"

	_, err := m.client.StatObject(ctx, m.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (m *MinioClient) DownloadFile(ctx context.Context, hash string, outputDir string) error {
	objectName := hash + ".zip"

	outputPath := filepath.Join(outputDir, objectName)

	err := m.client.FGetObject(
		ctx,
		m.bucketName,
		objectName,
		outputPath,
		minio.GetObjectOptions{},
	)

	if err != nil {
		return err
	}

	slog.Info("Download success:", "path", outputPath)
	return nil
}