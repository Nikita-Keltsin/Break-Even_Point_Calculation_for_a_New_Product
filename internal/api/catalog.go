package api

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CatalogPage struct {
	Items    []BreakEvenPointListJSON `json:"items"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

func queryInt(ctx *gin.Context, name string, fallback, max int) (int, bool) {
	raw, present := ctx.GetQuery(name)
	if !present {
		return fallback, true
	}
	n, err := strconv.Atoi(raw)
	return n, err == nil && n >= 1 && n <= max
}

func queryAmount(ctx *gin.Context, name string) (*float64, bool) {
	raw, present := ctx.GetQuery(name)
	if !present {
		return nil, true
	}
	n, err := strconv.ParseFloat(raw, 64)
	return &n, err == nil && !math.IsNaN(n) && !math.IsInf(n, 0) && n >= 0
}

// Catalog godoc
// @Summary      Постраничный каталог опубликованных видов затрат
// @Tags         breakeven_point
// @Produce      json
// @Param        min_amount query number false "Сумма от, ₽/мес"
// @Param        max_amount query number false "Сумма до, ₽/мес"
// @Param        is_fixed query int false "Тип затрат" Enums(0, 1)
// @Param        page query int false "Номер страницы" default(1)
// @Param        page_size query int false "Карточек на странице" default(24) maximum(100)
// @Success      200 {object} CatalogPage
// @Failure      400 "некорректные параметры каталога"
// @Router       /api/breakeven_point/catalog [get]
func (a *API) Catalog(ctx *gin.Context) {
	page, validPage := queryInt(ctx, "page", 1, 1000000)
	size, validSize := queryInt(ctx, "page_size", 24, 100)
	from, validFrom := queryAmount(ctx, "min_amount")
	to, validTo := queryAmount(ctx, "max_amount")
	if !validPage || !validSize || !validFrom || !validTo || (from != nil && to != nil && *from > *to) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid catalog parameters"})
		return
	}
	var fixed *bool
	if raw, present := ctx.GetQuery("is_fixed"); present {
		switch raw {
		case "0", "1", "false", "true":
			x := raw == "1" || raw == "true"
			fixed = &x
		default:
			ctx.Status(http.StatusBadRequest)
			return
		}
	}
	costs, total, err := a.Repo.Catalog(ctx.Request.Context(), from, to, fixed, page, size)
	if err != nil {
		ctx.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	uid := currentUserID(ctx)
	response := CatalogPage{Items: make([]BreakEvenPointListJSON, 0, len(costs)), Total: total, Page: page, PageSize: size}
	for _, c := range costs {
		response.Items = append(response.Items, BreakEvenPointListJSON{
			BreakEvenPointJSON: a.costJSON(c.BreakEvenPoint, c.LikesCount),
			IsMine:             boolTo01(uid != 0 && c.CreatorID == uid),
		})
	}
	ctx.JSON(http.StatusOK, response)
}
