package api

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

func validGLB(file multipart.File, header *multipart.FileHeader) bool {
	var magic [12]byte
	if !strings.EqualFold(path.Ext(header.Filename), ".glb") || header.Size < 20 || header.Size > 20<<20 {
		return false
	}
	if _, err := io.ReadFull(file, magic[:]); err != nil {
		return false
	}
	_, err := file.Seek(0, io.SeekStart)
	return err == nil && string(magic[:4]) == "glTF" && binary.LittleEndian.Uint32(magic[4:8]) == 2 &&
		int64(binary.LittleEndian.Uint32(magic[8:])) == header.Size
}

func (a *API) uploadCostMedia(ctx *gin.Context, uid int) (keys map[string]string, err error) {
	keys = make(map[string]string)
	defer func() {
		if err != nil {
			a.removeCostMedia(keys)
		}
	}()
	for _, kind := range []string{"image", "model", "video"} {
		file, header, openErr := ctx.Request.FormFile(kind)
		if openErr != nil {
			if openErr == http.ErrMissingFile {
				continue
			}
			return keys, fmt.Errorf("не удалось открыть %s", kind)
		}
		contentType := header.Header.Get("Content-Type")
		if kind == "model" {
			if !validGLB(file, header) {
				file.Close()
				return keys, fmt.Errorf("загрузите файл GLB версии 2 размером до 20 МиБ")
			}
			contentType = "model/gltf-binary"
		}
		key := latinName(uid, kind, header.Filename)
		_, uploadErr := a.MC.PutObject(ctx.Request.Context(), BucketName, key, file, header.Size, minio.PutObjectOptions{ContentType: contentType})
		file.Close()
		if uploadErr != nil {
			ctx.Error(uploadErr)
			return keys, fmt.Errorf("не удалось сохранить %s", kind)
		}
		keys[kind] = key
	}
	return keys, nil
}

func (a *API) removeCostMedia(keys map[string]string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, key := range keys {
		_ = a.MC.RemoveObject(ctx, BucketName, key, minio.RemoveObjectOptions{})
	}
}
