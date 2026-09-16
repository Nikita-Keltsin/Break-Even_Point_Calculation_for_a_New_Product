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

// общие данные для шаблонов: черновик + активная вкладка
func (h *Handler) baseH(page string) gin.H {
	hasDraft, draftID, draftCount := false, 0, 0
	if h.Repository.Request != nil && h.Repository.Request.Status == "черновик" {
		hasDraft, draftID, draftCount = true, h.Repository.Request.ID, len(h.Repository.Request.Links)
	}
	return gin.H{
		"Page":       page,
		"HasDraft":   hasDraft,
		"DraftID":    draftID,
		"DraftCount": draftCount,
		"MinIO":      h.MinIOBase,
	}
}

// ЛЕНТА
func (h *Handler) FeedHandler(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	cost, err := h.Repository.GetCostTypeByID(id)
	if err != nil {
		ctx.String(http.StatusNotFound, err.Error())
		return
	}
	data := h.baseH("feed")
	data["Cost"] = cost
	data["NextID"] = h.Repository.GetNextCostID(id)
	data["Full"] = ctx.Query("full") == "1"
	ctx.HTML(http.StatusOK, "feed.html", data)
}

// ЛАЙК
func (h *Handler) LikeHandler(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	h.Repository.ToggleLike(id)
	loc := fmt.Sprintf("/breakeven-point/feed/%d", id)
	if ctx.Query("full") == "1" {
		loc += "?full=1"
	}
	ctx.Redirect(http.StatusFound, loc)
}

// ДОБАВЛЕНИЕ (форма)
func (h *Handler) AdditionHandler(ctx *gin.Context) {
	data := h.baseH("addition")
	data["Images"] = h.Repository.MediaImages
	data["Videos"] = h.Repository.MediaVideos
	data["SelImage"] = h.Repository.MediaImages[0]
	data["SelVideo"] = h.Repository.MediaVideos[0]
	ctx.HTML(http.StatusOK, "addition.html", data)
}

// СОЗДАНИЕ карточки вида затрат
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
	})
	ctx.Redirect(http.StatusFound, "/breakeven-point/cost-types")
}

// ВИДЫ ЗАТРАТ: поиск/фильтр + карточка корзины
func (h *Handler) CostTypesHandler(ctx *gin.Context) {
	search := ctx.Query("search")
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

	costs := h.Repository.GetCostTypesFiltered(from, to, hasFrom, hasTo, fixedOn, varOn)
	if search != "" {
		var filtered []repository.CostType
		for _, c := range costs {
			if strings.Contains(strings.ToLower(c.CostName), strings.ToLower(search)) {
				filtered = append(filtered, c)
			}
		}
		costs = filtered
	}

	data := h.baseH("costs")
	data["Costs"] = costs
	data["Search"] = search
	data["Preset"] = preset
	data["From"] = fromStr
	data["To"] = toStr
	data["FixedOn"] = fixedOn
	data["VarOn"] = varOn
	ctx.HTML(http.StatusOK, "cost_types.html", data)
}

// ЗАЯВКА: состав + расчёт ТБУ по формуле; удалённые не просматриваются
func (h *Handler) RequestHandler(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	req, err := h.Repository.GetRequestByID(id)
	if err != nil {
		ctx.String(http.StatusNotFound, "Заявка не найдена")
		return
	}
	if req.Status == "deleted" {
		ctx.String(http.StatusForbidden, "Заявка удалена: просмотр запрещён")
		return
	}
	if req.BepUnits == 0 {
		units, revenue := h.Repository.ComputeBreakEven(req)
		req.BepUnits, req.BepRevenue = units, revenue
	}
	data := h.baseH("request")
	data["Request"] = req
	ctx.HTML(http.StatusOK, "request.html", data)
}

// ДОБАВИТЬ услугу в заявку (карточка корзины меняется)
func (h *Handler) AddToRequestHandler(ctx *gin.Context) {
	costID, _ := strconv.Atoi(ctx.PostForm("cost_id"))
	volume, _ := strconv.Atoi(ctx.PostForm("volume"))
	if volume == 0 {
		volume = 1
	}
	if err := h.Repository.AddCostToRequest(costID, volume, ""); err != nil {
		log.Println(err)
	}
	ctx.Redirect(http.StatusFound, "/breakeven-point/cost-types")
}

// ПРАВКА поля м-м (объём)
func (h *Handler) UpdateLinkHandler(ctx *gin.Context) {
	costID, _ := strconv.Atoi(ctx.PostForm("cost_id"))
	volume, _ := strconv.Atoi(ctx.PostForm("volume"))
	if err := h.Repository.UpdateLinkVolume(costID, volume); err != nil {
		log.Println(err)
	}
	ctx.Redirect(http.StatusFound, "/breakeven-point/request/"+ctx.PostForm("request_id"))
}

// ЛОГИЧЕСКОЕ УДАЛЕНИЕ заявки
func (h *Handler) DeleteRequestHandler(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	if err := h.Repository.DeleteRequest(id); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Redirect(http.StatusFound, "/breakeven-point/cost-types")
}

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