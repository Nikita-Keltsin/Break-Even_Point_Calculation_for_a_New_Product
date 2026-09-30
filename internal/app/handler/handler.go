package handler

import (
	"log"
	"net/http"
	"strconv"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"

	"github.com/gin-gonic/gin"
)

func CurrentUserID() int { return 1 }

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler { return &Handler{Repository: r} }

type CostCard struct {
	Cost       repository.CostType
	ImageURL   string
	LikesCount int64
}

// ЛЕНТА: id берём из :id или ?id=, full — из ?full=1
func (h *Handler) FeedHandler(ctx *gin.Context) {
	idStr := ctx.Param("id")
	if idStr == "" {
		idStr = ctx.Query("id")
	}
	id, _ := strconv.Atoi(idStr)
	cost, err := h.Repository.GetFeedCost(id)
	if err != nil {
		ctx.String(http.StatusNotFound, "Услуга не найдена или была удалена")
		return
	}
	ctx.HTML(http.StatusOK, "feed.html", gin.H{
		"Cost":     cost,
		"VideoURL": h.Repository.VideoURL(cost),
		"Likes":    h.Repository.GetLikesCount(cost.ID),
		"Liked":    h.Repository.HasLiked(CurrentUserID(), cost.ID),
		"NextID":   h.Repository.GetNextPublishedID(id),
		"Full":     ctx.Query("full") == "1",
		"Page":     "feed",
	})
}

// ЛАЙК: редирект обратно в том же стиле маршрута
func (h *Handler) LikeHandler(ctx *gin.Context) {
	costID, _ := strconv.Atoi(ctx.PostForm("cost_id"))
	if err := h.Repository.ToggleLike(CurrentUserID(), costID); err != nil {
		log.Println(err)
	}
	loc := "/breakeven-point/feed/" + ctx.PostForm("cost_id")
	if ctx.PostForm("full") == "1" {
		loc += "?full=1"
	}
	ctx.Redirect(http.StatusFound, loc)
}

// ДОБАВЛЕНИЕ: превью из resources/media (SSR), НЕ из MinIO
func (h *Handler) AdditionHandler(ctx *gin.Context) {
	draft, err := h.Repository.GetDraftByUser(CurrentUserID())
	ctx.HTML(http.StatusOK, "addition.html", gin.H{
		"HasDraft": err == nil,
		"Draft":    draft,
		"Page":     "addition",
	})
}

func (h *Handler) CreateDraftHandler(ctx *gin.Context) {
	name := ctx.PostForm("cost_name")
	if name == "" {
		ctx.String(http.StatusBadRequest, "Название обязательно")
		return
	}
	if _, err := h.Repository.GetDraftByUser(CurrentUserID()); err == nil {
		ctx.Redirect(http.StatusFound, "/breakeven-point/addition")
		return
	}
	kind := ctx.PostForm("cost_kind")
	var amt *float64
	if s := ctx.PostForm("amount_monthly"); s != "" {
		v, _ := strconv.ParseFloat(s, 64)
		amt = &v
	}
	err := h.Repository.CreateDraft(CurrentUserID(), repository.CostType{
		CostName:         name,
		ShortDescription: ctx.PostForm("description"),
		FullDescription:  ctx.PostForm("full_description"),
		CostKind:         &kind,
		AmountMonthly:    amt,
	})
	if err != nil {
		log.Println(err)
	}
	ctx.Redirect(http.StatusFound, "/breakeven-point/addition")
}

func (h *Handler) PublishHandler(ctx *gin.Context) {
	amount, _ := strconv.ParseFloat(ctx.PostForm("amount_monthly"), 64)
	err := h.Repository.PublishDraft(CurrentUserID(),
		ctx.PostForm("description"),
		ctx.PostForm("full_description"),
		amount,
		ctx.PostForm("cost_kind"))
	if err != nil {
		log.Println(err)
	}
	ctx.Redirect(http.StatusFound, "/breakeven-point/cost-types")
}

func (h *Handler) CostTypesHandler(ctx *gin.Context) {
	fromStr := ctx.Query("amount_from")
	toStr := ctx.Query("amount_to")
	from, err1 := strconv.ParseFloat(fromStr, 64)
	to, err2 := strconv.ParseFloat(toStr, 64)
	hasFrom, hasTo := err1 == nil && fromStr != "", err2 == nil && toStr != ""
	if hasFrom && hasTo && from > to {
		from, to = to, from
		fromStr = strconv.FormatFloat(from, 'f', -1, 64)
		toStr = strconv.FormatFloat(to, 'f', -1, 64)
	}
	if fromStr == "" {
		fromStr = "0"
	}
	if toStr == "" {
		toStr = "8000000"
	}
	fixedOn := ctx.Query("type_fixed") == "on"
	varOn := ctx.Query("type_variable") == "on"

	costs, err := h.Repository.GetPublishedCosts(from, to, hasFrom, hasTo, fixedOn, varOn)
	if err != nil {
		log.Println(err)
	}
	var cards []CostCard
	for _, c := range costs {
		cards = append(cards, CostCard{
			Cost:       c,
			ImageURL:   h.Repository.ImageURL(c),
			LikesCount: h.Repository.GetLikesCount(c.ID),
		})
	}
	ctx.HTML(http.StatusOK, "cost_types.html", gin.H{
		"Costs":    cards,
		"From":     fromStr,
		"To":       toStr,
		"FromText": repository.CostType{AmountMonthly: &from}.AmountText(),
		"ToText":   repository.CostType{AmountMonthly: &to}.AmountText(),
		"FixedOn":  fixedOn,
		"VarOn":    varOn,
		"Page":     "costs",
	})
}

func (h *Handler) DeleteHandler(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.PostForm("id"))
	if err := h.Repository.DeleteCostSQL(id); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Redirect(http.StatusFound, "/breakeven-point/cost-types")
}