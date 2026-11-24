package api

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"
	"github.com/gin-gonic/gin"
)

const demoRotationKey = "rotate08"

func setting(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

var rotationClient = &http.Client{Timeout: 2 * time.Second}

func (a *API) queueRotation(ctx context.Context, id int) error {
	body, err := json.Marshal(map[string]int{"id": id})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, setting("ROTATION_URL", "http://127.0.0.1:3100")+"/rotations", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Rotation-Key", setting("ROTATION_KEY", demoRotationKey))
	res, err := rotationClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		return fmt.Errorf("rotation service: %s", res.Status)
	}
	return nil
}

// SaveRotation godoc
// @Summary      Сохранить результат поворота 3D-модели
// @Tags         rotation
// @Param        id path int true "ID вида затрат"
// @Param        X-Rotation-Key header string true "Ключ межсервисного вызова"
// @Success      204 "результат сохранён"
// @Failure      400 "некорректные поля"
// @Failure      401 "неверный ключ"
// @Failure      409 "версия изменилась или карточка недоступна"
// @Router       /internal/costs/{id}/rotation [put]
func (a *API) SaveRotation(ctx *gin.Context) {
	if subtle.ConstantTimeCompare([]byte(ctx.GetHeader("X-Rotation-Key")), []byte(setting("ROTATION_KEY", demoRotationKey))) != 1 {
		ctx.Status(http.StatusUnauthorized)
		return
	}
	id, err := strconv.Atoi(ctx.Param("id"))
	var body struct {
		X       *int `json:"angle_x"`
		Y       *int `json:"angle_y"`
		Version *int `json:"rotation_version"`
	}
	if err != nil || id < 1 || ctx.ShouldBindJSON(&body) != nil || body.X == nil || body.Y == nil ||
		*body.X < 0 || *body.X >= 360 || *body.Y < 0 || *body.Y >= 360 ||
		body.Version == nil || *body.Version < 0 {
		ctx.Status(http.StatusBadRequest)
		return
	}
	result := a.Repo.DB.WithContext(ctx.Request.Context()).Model(&repository.BreakEvenPoint{}).
		Where("id = ? AND status = 'published' AND rotation_version = ?", id, *body.Version).
		Updates(map[string]any{"angle_x": *body.X, "angle_y": *body.Y, "rotation_version": *body.Version + 1})
	switch {
	case result.Error != nil:
		ctx.Error(result.Error)
		ctx.Status(http.StatusInternalServerError)
	case result.RowsAffected == 0:
		ctx.Status(http.StatusConflict)
	default:
		ctx.Status(http.StatusNoContent)
	}
}
