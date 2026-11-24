package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"
)

const feedTTL = 24 * time.Hour

var sessionIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type feedBatch struct {
	IDs        []int `json:"ids"`
	Position   int   `json:"position"`
	PreviousID int   `json:"previous_id"`
	Number     int   `json:"batch"`
}

func (a *API) feedKey(ctx *gin.Context) string {
	id := ctx.GetHeader("X-Feed-Session")
	if !sessionIDPattern.MatchString(id) {
		id, _ = ctx.Cookie("feed_session")
	}
	if !sessionIDPattern.MatchString(id) {
		var bytes [16]byte
		_, _ = rand.Read(bytes[:])
		id = hex.EncodeToString(bytes[:])
		ctx.SetSameSite(http.SameSiteLaxMode)
		ctx.SetCookie("feed_session", id, int(feedTTL.Seconds()), "/", "", ctx.Request.TLS != nil || ctx.GetHeader("X-Forwarded-Proto") == "https", true)
	}
	return fmt.Sprintf("feed:%d:%s", currentUserID(ctx), id)
}

func (a *API) loadFeedBatch(ctx context.Context, key string, after *int) (feedBatch, error) {
	var batch feedBatch
	for attempt := 0; attempt < 8; attempt++ {
		err := a.RDB.Watch(ctx, func(tx *redis.Tx) error {
			batch = feedBatch{}
			raw, err := tx.Get(ctx, key).Bytes()
			if err == nil {
				if err = json.Unmarshal(raw, &batch); err != nil {
					return err
				}
			} else if !errors.Is(err, redis.Nil) {
				return err
			}
			if len(batch.IDs) == 0 {
				batch.IDs, err = a.Repo.RandomCostIDs(ctx, nil)
				if err != nil {
					return err
				}
				batch.Number = 1
			} else if after == nil {
				return nil
			} else {
				// Requests from an old screen replay the current card; only the
				// current cursor advances. A direct link outside the batch joins
				// at the active card without changing the order.
				if *after != batch.IDs[batch.Position] || (*after == batch.PreviousID && len(batch.IDs) == 1) {
					return nil
				}
				batch.PreviousID = *after
				batch.Position++
				if batch.Position == len(batch.IDs) {
					batch.IDs, err = a.Repo.RandomCostIDs(ctx, batch.IDs)
					if err != nil {
						return err
					}
					batch.Position = 0
					batch.Number++
				}
			}
			raw, err = json.Marshal(batch)
			if err != nil {
				return err
			}
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, raw, feedTTL)
				return nil
			})
			return err
		}, key)
		if !errors.Is(err, redis.TxFailedErr) {
			return batch, err
		}
	}
	return batch, redis.TxFailedErr
}

// InitFeed godoc
// @Summary      Подготовить порцию сессионной ленты
// @Tags         breakeven_point
// @Produce      json
// @Param        X-Feed-Session header string false "Идентификатор сессии (32 hex-символа)"
// @Success      200 {object} map[string]int
// @Failure      503 "хранилище недоступно"
// @Router       /api/breakeven_point/feed/session [get]
func (a *API) InitFeed(ctx *gin.Context) {
	batch, err := a.loadFeedBatch(ctx.Request.Context(), a.feedKey(ctx), nil)
	if err != nil {
		ctx.Error(err)
		ctx.Status(http.StatusServiceUnavailable)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, gin.H{"batch": batch.Number, "size": len(batch.IDs)})
}

func (a *API) sessionCost(ctx *gin.Context, after *int) (repository.BreakEvenPoint, feedBatch, error) {
	key := a.feedKey(ctx)
	batch, err := a.loadFeedBatch(ctx.Request.Context(), key, after)
	for skipped := 0; err == nil && skipped < 100 && len(batch.IDs) > 0; skipped++ {
		id := batch.IDs[batch.Position]
		cost, readErr := a.Repo.GetFeedCost(id)
		if !errors.Is(readErr, gorm.ErrRecordNotFound) {
			return cost, batch, readErr
		}
		batch, err = a.loadFeedBatch(ctx.Request.Context(), key, &id)
	}
	if err == nil {
		err = gorm.ErrRecordNotFound
	}
	return repository.BreakEvenPoint{}, batch, err
}

func batchPosition(batch feedBatch, id int) int {
	return slices.Index(batch.IDs, id) + 1
}
