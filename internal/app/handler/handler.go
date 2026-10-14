package handler

import (
	"log"
	"net/http"
	"strconv"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"

	"github.com/gin-gonic/gin"
)

// ЛР-2 (SSR): 6 HTTP-методов
//   GET  /breakeven-point/cost-types  — плитка услуг + поиск (ORM)
//   GET  /breakeven-point/feed[/:id]  — лента, одна строка из БД (ORM)
//   GET  /breakeven-point/addition    — добавление (ORM)
//   POST /breakeven-point/draft       — «Далее»: создать черновик (ORM)
//   POST /breakeven-point/publish     — «Опубликовать» (ORM)
//   POST /breakeven-point/delete      — логическое удаление (SQL UPDATE, без ORM)
// Лайки во 2-й лабораторной только отображаются из БД.

func CurrentUserID() int { return 1 }

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler { return &Handler{Repository: r} }

type CostCard struct {
	Cost       repository.BreakEvenType
	ImageURL   string
	LikesCount int64
}

// GET /breakeven-point/cost-types — плитка + фильтр по сумме и виду затрат
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
		to = 8000000
	}
	fixedOn := ctx.Query("type_fixed") == "on"
	varOn := ctx.Query("type_variable") == "on"
	kind := ""
	if fixedOn && !varOn {
		kind = "fixed"
	} else if varOn && !fixedOn {
		kind = "variable"
	}

	costs, err := h.Repository.GetPublishedCosts(from, to, hasFrom, hasTo, kind)
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
		"FromText": repository.BreakEvenType{AmountMonthly: &from}.AmountText(),
		"ToText":   repository.BreakEvenType{AmountMonthly: &to}.AmountText(),
		"FixedOn":  fixedOn,
		"VarOn":    varOn,
		"Page":     "costs",
	})
}

// GET /breakeven-point/feed и /breakeven-point/feed/:id
func (h *Handler) FeedHandler(ctx *gin.Context) {
	idStr := ctx.Param("id")
	if idStr == "" {
		idStr = ctx.Query("id")
	}
	id, _ := strconv.Atoi(idStr)
	cost, err := h.Repository.GetFeedCost(id) // одна строка из БД
	if err != nil {
		ctx.String(http.StatusNotFound, "Услуга не найдена или была удалена")
		return
	}
	next, err := h.Repository.GetNextFeedCost(cost.ID)
	nextID := cost.ID
	if err == nil {
		nextID = next.ID
	}
	ctx.HTML(http.StatusOK, "feed.html", gin.H{
		"Cost":     cost,
		"VideoURL": h.Repository.VideoURL(cost),
		"Likes":    h.Repository.GetLikesCount(cost.ID),
		"Liked":    h.Repository.HasLiked(CurrentUserID(), cost.ID),
		"NextID":   nextID,
		"Page":     "feed",
	})
}

// GET /breakeven-point/addition
// Нет черновика → название, фото, видео и кнопка «Далее».
// Есть черновик → поля заполнены + краткое описание, 2 поля по теме и кнопка «Опубликовать».
func (h *Handler) AdditionHandler(ctx *gin.Context) {
	draft, err := h.Repository.GetDraftByUser(CurrentUserID())
	data := gin.H{
		"HasDraft": err == nil,
		"Page":     "addition",
		"ImageURL": repository.DefaultImageURL,
		"VideoURL": repository.DefaultVideoURL,
	}
	if err == nil {
		data["Draft"] = draft
		data["ImageURL"] = h.Repository.ImageURL(*draft)
		data["VideoURL"] = h.Repository.VideoURL(*draft)
	}
	ctx.HTML(http.StatusOK, "addition.html", data)
}

// POST /breakeven-point/draft — «Далее»: черновик создаётся только здесь (ORM)
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
	if err := h.Repository.CreateDraft(CurrentUserID(), name); err != nil {
		log.Println(err)
	}
	ctx.Redirect(http.StatusFound, "/breakeven-point/addition")
}

// POST /breakeven-point/publish — «Опубликовать» (ORM)
func (h *Handler) PublishHandler(ctx *gin.Context) {
	description := ctx.PostForm("short_description")
	amount, err := strconv.ParseFloat(ctx.PostForm("amount_monthly"), 64)
	if description == "" || err != nil {
		ctx.String(http.StatusBadRequest, "Заполните краткое описание и сумму затрат")
		return
	}
	isFixed := ctx.PostForm("is_fixed") == "on" // галочка
	if err := h.Repository.PublishDraft(CurrentUserID(), description, amount, isFixed); err != nil {
		log.Println(err)
	}
	ctx.Redirect(http.StatusFound, "/breakeven-point/cost-types")
}

// POST /breakeven-point/delete — логическое удаление через SQL UPDATE (без ORM)
func (h *Handler) DeleteHandler(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.PostForm("id"))
	if err := h.Repository.DeleteCostSQL(id); err != nil {
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Redirect(http.StatusFound, "/breakeven-point/cost-types")
}
