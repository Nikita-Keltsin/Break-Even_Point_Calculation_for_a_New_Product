package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Repository *repository.Repository
	MinIOBase  string
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{Repository: r, MinIOBase: "http://localhost:9000/media/"}
}

// ЛЕНТА: видео из MinIO + «Еще» (?full=1) раскрывает полное описание прямо на экране
func (h *Handler) FeedHandler(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		log.Println(err)
	}
	cost, err := h.Repository.GetCostTypeByID(id)
	if err != nil {
		ctx.String(http.StatusNotFound, err.Error())
		return
	}
	ctx.HTML(http.StatusOK, "feed.html", gin.H{
		"Cost":   cost,
		"NextID": h.Repository.GetNextCostID(id),
		"Full":   ctx.Query("full") == "1",
		"MinIO":  h.MinIOBase,
	})
}

// ЛАЙК: сердечко загорается, счётчик +1 (повторно — гаснет)
func (h *Handler) LikeHandler(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	h.Repository.ToggleLike(id)
	loc := fmt.Sprintf("/breakeven-point/feed/%d", id)
	if ctx.Query("full") == "1" {
		loc += "?full=1"
	}
	ctx.Redirect(http.StatusFound, loc)
}

// ДОБАВЛЕНИЕ: превью фото/видео из MinIO плиткой + модальные окна выбора
func (h *Handler) AdditionHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "addition.html", gin.H{
		"Images":   h.Repository.MediaImages,
		"Videos":   h.Repository.MediaVideos,
		"SelImage": h.Repository.MediaImages[0],
		"SelVideo": h.Repository.MediaVideos[0],
		"MinIO":    h.MinIOBase, // ← вот этой строки не хватало: без неё src собирался относительным и ломался
	})
}

// СОХРАНИТЬ ЗАЯВКУ = создать новую карточку в «Видах затрат»
func (h *Handler) AdditionSubmitHandler(ctx *gin.Context) {
	name := ctx.PostForm("cost_name")
	if name == "" {
		name = "Новая статья затрат"
	}
	desc := ctx.PostForm("description")
	img := ctx.PostForm("image_key")
	if img == "" {
		img = h.Repository.MediaImages[0]
	}
	vid := ctx.PostForm("video_key")
	if vid == "" {
		vid = h.Repository.MediaVideos[0]
	}
	h.Repository.AddCostType(repository.CostType{
		CostName:         name,
		ShortDescription: desc,
		FullDescription:  desc,
		ImageKey:         img,
		VideoKey:         vid,
		CostKind:         ctx.PostForm("cost_kind"),
		AmountMonthly:    parseAmount(ctx.PostForm("amount")),
		CostShare:        0,
	})
	ctx.Redirect(http.StatusFound, "/breakeven-point/cost-types")
}

// ВИДЫ ЗАТРАТ: рабочий фильтр по сумме (пресеты + диапазон) и галочкам
func (h *Handler) CostTypesHandler(ctx *gin.Context) {
	preset := ctx.Query("preset")
	fromStr := ctx.Query("amount_from")
	toStr := ctx.Query("amount_to")

	var from, to float64
	hasFrom, hasTo := false, false
	switch preset {
	case "lt50":
		to, hasTo, toStr = 50000, true, "50000"
	case "b50_100":
		from, hasFrom, to, hasTo = 50000, true, 100000, true
		fromStr, toStr = "50000", "100000"
	case "gt100":
		from, hasFrom, fromStr = 100000, true, "100000"
	default:
		if v, err := strconv.ParseFloat(fromStr, 64); err == nil && fromStr != "" {
			from, hasFrom = v, true
		}
		if v, err := strconv.ParseFloat(toStr, 64); err == nil && toStr != "" {
			to, hasTo = v, true
		}
	}
	fixedOn := ctx.Query("type_fixed") == "on"
	varOn := ctx.Query("type_variable") == "on"

	ctx.HTML(http.StatusOK, "cost_types.html", gin.H{
		"Costs":   h.Repository.GetCostTypesFiltered(from, to, hasFrom, hasTo, fixedOn, varOn),
		"Preset":  preset,
		"From":    fromStr,
		"To":      toStr,
		"FixedOn": fixedOn,
		"VarOn":   varOn,
		"MinIO":   h.MinIOBase,
	})
}

// parseAmount — вытаскивает число из строки вида «150 000 ₽ / 15% / мес»
func parseAmount(s string) float64 {
	var sb strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			sb.WriteRune(r)
		}
	}
	v, _ := strconv.ParseFloat(sb.String(), 64)
	return v
}