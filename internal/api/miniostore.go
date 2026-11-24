package api

import (
	"context"
	"fmt"
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
	return minio.New(setting("MINIO_ENDPOINT", minioEndpoint), &minio.Options{
		Creds: credentials.NewStaticV4(setting("MINIO_USER", "root"), setting("MINIO_PASSWORD", "rootpassword"), ""),
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

// Имя файла генерируется латиницей: cost_<uid>_<image|video>_<unixnano>.<ext>
func latinName(uid int, kind, orig string) string {
	ext := strings.ToLower(path.Ext(orig))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".mp4", ".webm", ".mov", ".glb":
	default:
		if kind == "video" {
			ext = ".mp4"
		} else {
			ext = ".jpg"
		}
	}
	return fmt.Sprintf("cost_%d_%s_%d%s", uid, kind, time.Now().UnixNano(), ext)
}
