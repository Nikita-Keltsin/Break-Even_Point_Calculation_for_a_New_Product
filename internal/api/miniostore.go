package api

import (
	"context"
	"fmt"
	"mime/multipart"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	minioEndpoint = "localhost:9000"
	BucketName    = "break-even"
)

func NewMinIOClient() (*minio.Client, error) {
	return minio.New(minioEndpoint, &minio.Options{
		Creds: credentials.NewStaticV4("root", "rootpassword", ""),
	})
}

func EnsureBucket(mc *minio.Client) error {
	ctx := context.Background()
	exists, err := mc.BucketExists(ctx, BucketName)
	if err != nil {
		return err
	}
	if !exists {
		if err := mc.MakeBucket(ctx, BucketName, minio.MakeBucketOptions{}); err != nil {
			return err
		}
		policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::` + BucketName + `/*"]}]}`
		return mc.SetBucketPolicy(ctx, BucketName, policy)
	}
	return nil
}

// имя файла — только латиница: cost_<uid>_<unix>.<ext>
func latinName(uid int, orig string) string {
	ext := strings.ToLower(path.Ext(orig))
	switch ext {
	case ".jpeg", ".png", ".webp", ".jpg":
		ext = ".jpg"
	case ".mp4", ".webm", ".mov":
		ext = ".mp4"
	default:
		ext = ".bin"
	}
	return fmt.Sprintf("cost_%d_%d%s", uid, time.Now().Unix(), ext)
}

func UploadFile(mc *minio.Client, uid int, file multipart.File, orig, contentType string) (string, error) {
	key := latinName(uid, orig)
	_, err := mc.PutObject(context.Background(), BucketName, key, file, -1, minio.PutObjectOptions{ContentType: contentType})
	return key, err
}