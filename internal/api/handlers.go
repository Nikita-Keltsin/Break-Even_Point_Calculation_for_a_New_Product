package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// ЛР-4: функция-singleton CurrentUserID() из ЛР-3 удалена — пользователь берётся из JWT: currentUserID(ctx) в auth.go

type API struct {
	Repo *repository.Repository
	MC   *minio.Client
	RDB  *redis.Client // ЛР-4: blacklist отозванных JWT
}

func boolTo01(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (a *API) fullURL(key string) string {
	if key == "" {
		return ""
	}
	if strings.HasPrefix(key, "http") {
		return key
	}
	return "http://" + minioEndpoint + "/" + BucketName + "/" + key
}

// toCostJSON — uid текущего пользователя (0 — гость) нужен для полей like и is_mine
func (a *API) toCostJSON(c repository.BreakEvenType, uid int) BreakEvenTypeJSON {
	return BreakEvenTypeJSON{
		ID:               c.ID,
		CostName:         c.CostName,
		ShortDescription: c.ShortDescription,
		ImageURL:         a.fullURL(c.ImageURL),
		VideoURL:         a.fullURL(c.VideoURL),
		AmountMonthly:    c.AmountMonthly,
		IsFixed:          c.IsFixed,
		CreatorID:        c.CreatorID,
		CreatedAt:        c.CreatedAt,
		FormedAt:         c.FormedAt,
		LikesCount:       a.Repo.GetLikesCount(c.ID),
		Like:             boolTo01(a.Repo.HasLiked(uid, c.ID)),
		IsMine:           boolTo01(uid != 0 && c.CreatorID == uid),
	}
}

// ===================== ДОМЕН «УСЛУГА» (/api/breakeven_point) =====================

// 1. GET /api/breakeven_point?min_amount=&max_amount=&is_fixed=0|1 — список опубликованных, фильтр на бэкенде
//
// ListCosts godoc
// @Summary      Список видов затрат с фильтром
// @Description  Только опубликованные. Фильтрация на бэкенде. Доступно всем; с токеном заполняются like и is_mine.
// @Tags         breakeven_point
// @Produce      json
// @Param        min_amount query number false "Сумма от, ₽/мес"
// @Param        max_amount query number false "Сумма до, ₽/мес"
// @Param        is_fixed   query int    false "1 — постоянные, 0 — переменные" Enums(0, 1)
// @Success      200 {array} BreakEvenTypeJSON
// @Router       /api/breakeven_point [get]
func (a *API) ListCosts(ctx *gin.Context) {
	from, err1 := strconv.ParseFloat(ctx.Query("min_amount"), 64)
	to, err2 := strconv.ParseFloat(ctx.Query("max_amount"), 64)
	hasFrom := err1 == nil && ctx.Query("min_amount") != ""
	hasTo := err2 == nil && ctx.Query("max_amount") != ""
	kind := ""
	switch ctx.Query("is_fixed") {
	case "1", "true":
		kind = "fixed"
	case "0", "false":
		kind = "variable"
	}
	costs, err := a.Repo.GetPublishedCosts(from, to, hasFrom, hasTo, kind)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	res := make([]BreakEvenTypeJSON, 0, len(costs))
	for _, c := range costs {
		res = append(res, a.toCostJSON(c, currentUserID(ctx)))
	}
	ctx.JSON(http.StatusOK, res)
}

//  2. GET /api/breakeven_point/feed            — первая опубликованная (без id)
//     GET /api/breakeven_point/feed/:id        — опубликованная по id
//     GET /api/breakeven_point/feed/:id?next=1 — следующая после id (next=true тоже принимается)
//
// Из БД всегда читается одна строка; в ответе like=0/1 для текущего пользователя.
//
// FeedCost godoc
// @Summary      Лента: один вид затрат
// @Description  Без id — первый опубликованный; с id — по id; с id и next=true — следующий после id. Доступно всем.
// @Tags         breakeven_point
// @Produce      json
// @Param        id   query int  false "id вида затрат"
// @Param        next query bool false "true — следующий после id"
// @Success      200 {object} BreakEvenTypeJSON
// @Failure      404 "не найден или удалён"
// @Router       /api/breakeven_point/feed [get]
func (a *API) FeedCost(ctx *gin.Context) {
	idStr := ctx.Param("id")
	if idStr == "" {
		idStr = ctx.Query("id") // GET /api/breakeven_point/feed?id=1&next=true
	}
	id, _ := strconv.Atoi(idStr)
	var (
		c   repository.BreakEvenType
		err error
	)
	if next := ctx.Query("next"); id > 0 && (next == "1" || next == "true") {
		c, err = a.Repo.GetNextFeedCost(id)
	} else {
		c, err = a.Repo.GetFeedCost(id)
	}
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	ctx.JSON(http.StatusOK, a.toCostJSON(c, currentUserID(ctx)))
}

// 3. GET /api/breakeven_point/draft — черновик текущего пользователя (не более одного), id не передаётся
//
// GetDraft godoc
// @Summary      Черновик текущего пользователя
// @Description  Не более одного черновика у пользователя, id не передаётся. Доступно создателю.
// @Tags         breakeven_point
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} BreakEvenTypeJSON
// @Failure      401 "нет токена"
// @Failure      404 "черновика нет"
// @Router       /api/breakeven_point/draft [get]
func (a *API) GetDraft(ctx *gin.Context) {
	draft, err := a.Repo.GetDraftByUser(currentUserID(ctx))
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	ctx.JSON(http.StatusOK, a.toCostJSON(*draft, currentUserID(ctx)))
}

// 4. POST /api/breakeven_point — создание черновика: name + файлы image и video (multipart/form-data).
// Файлы кладутся в MinIO под латинскими именами, имена сохраняются в image_url/video_url.
//
// CreateCost godoc
// @Summary      Создать вид затрат (черновик) с фото и видео
// @Description  Файлы сохраняются в MinIO под латинскими именами; создатель — пользователь из JWT. Доступно создателю.
// @Tags         breakeven_point
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        name  formData string true  "Название"
// @Param        image formData file   false "Фото"
// @Param        video formData file   false "Видео"
// @Success      201 {object} BreakEvenTypeJSON
// @Failure      400 "нет названия"
// @Failure      401 "нет токена"
// @Failure      409 "черновик уже есть"
// @Router       /api/breakeven_point [post]
func (a *API) CreateCost(ctx *gin.Context) {
	uid := currentUserID(ctx)
	if _, err := a.Repo.GetDraftByUser(uid); err == nil {
		ctx.Status(http.StatusConflict) // не более одного черновика
		return
	}
	name := ctx.PostForm("name")
	if name == "" {
		ctx.Status(http.StatusBadRequest)
		return
	}
	imageKey, videoKey := "", ""
	if f, h, err := ctx.Request.FormFile("image"); err == nil {
		defer f.Close()
		if key, uerr := UploadFile(a.MC, uid, "image", f, h.Filename, h.Header.Get("Content-Type")); uerr == nil {
			imageKey = key
		}
	}
	if f, h, err := ctx.Request.FormFile("video"); err == nil {
		defer f.Close()
		if key, uerr := UploadFile(a.MC, uid, "video", f, h.Filename, h.Header.Get("Content-Type")); uerr == nil {
			videoKey = key
		}
	}
	c, err := a.Repo.CreateDraftWithMedia(uid, name, imageKey, videoKey)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.JSON(http.StatusCreated, a.toCostJSON(*c, uid))
}

// 5. PUT /api/breakeven_point/:id/publish — draft → published (только свой черновик; обратно нельзя)
//
// PublishCost godoc
// @Summary      Опубликовать свой черновик
// @Description  draft → published: краткое описание и 2 поля по теме, formed_at. Вернуть в черновик нельзя. Доступно создателю.
// @Tags         breakeven_point
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path int            true "id черновика"
// @Param        body body PublishRequest true "Краткое описание и 2 поля по теме"
// @Success      200 {object} BreakEvenTypeJSON
// @Failure      400 "неверное тело запроса"
// @Failure      401 "нет токена"
// @Failure      403 "чужая карточка или не черновик"
// @Failure      404 "не найдена"
// @Router       /api/breakeven_point/{id}/publish [put]
func (a *API) PublishCost(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	c, err := a.Repo.GetCostByID(id)
	if err != nil || c.Status == "deleted" {
		ctx.Status(http.StatusNotFound)
		return
	}
	if c.CreatorID != currentUserID(ctx) || c.Status != "draft" {
		ctx.Status(http.StatusForbidden)
		return
	}
	var body PublishRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	updates := map[string]interface{}{
		"short_description": body.ShortDescription,
		"amount_monthly":    *body.AmountMonthly,
		"is_fixed":          *body.IsFixed,
		"status":            "published",
		"formed_at":         time.Now(),
	}
	if err := a.Repo.UpdateCost(id, updates); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	c, _ = a.Repo.GetCostByID(id)
	ctx.JSON(http.StatusOK, a.toCostJSON(c, currentUserID(ctx)))
}

// 6. DELETE /api/breakeven_point/:id — только soft delete своей услуги (status = 'deleted', ORM)
//
// DeleteCost godoc
// @Summary      Удалить свой вид затрат (soft delete)
// @Tags         breakeven_point
// @Security     BearerAuth
// @Param        id path int true "id вида затрат"
// @Success      200 "статус deleted"
// @Failure      401 "нет токена"
// @Failure      403 "чужая карточка"
// @Failure      404 "не найдена"
// @Router       /api/breakeven_point/{id} [delete]
func (a *API) DeleteCost(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	c, err := a.Repo.GetCostByID(id)
	if err != nil || c.Status == "deleted" {
		ctx.Status(http.StatusNotFound)
		return
	}
	if c.CreatorID != currentUserID(ctx) {
		ctx.Status(http.StatusForbidden)
		return
	}
	if err := a.Repo.SoftDeleteCost(id); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.Status(http.StatusOK)
}

// 7. POST /api/breakeven_point/:id/like — {"like":1} ставит лайк, {"like":0} отменяет
//
// LikeCost godoc
// @Summary      Поставить или отменить лайк
// @Description  like = 1 ставит лайк текущего пользователя, like = 0 отменяет. Доступно создателю.
// @Tags         breakeven_point
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path int         true "id вида затрат"
// @Param        body body LikeRequest true "0 или 1"
// @Success      200 {object} LikeResponse
// @Failure      400 "like не 0 и не 1"
// @Failure      401 "нет токена"
// @Failure      404 "не найдена"
// @Router       /api/breakeven_point/{id}/like [post]
func (a *API) LikeCost(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	c, err := a.Repo.GetCostByID(id)
	if err != nil || c.Status != "published" {
		ctx.Status(http.StatusNotFound)
		return
	}
	var body LikeRequest
	if err := ctx.ShouldBindJSON(&body); err != nil || (*body.Like != 0 && *body.Like != 1) {
		ctx.Status(http.StatusBadRequest)
		return
	}
	if err := a.Repo.SetLike(currentUserID(ctx), id, *body.Like == 1); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.JSON(http.StatusOK, LikeResponse{Like: *body.Like, LikesCount: a.Repo.GetLikesCount(id)})
}

// ===================== ДОМЕН «ПОЛЬЗОВАТЕЛЬ» (/api/users) =====================

// 8. POST /api/users/register — {username, password} → 201 {id, username}; 409 если логин занят
//
// Register godoc
// @Summary      Регистрация пользователя
// @Description  Пароль хранится bcrypt-хэшем. Доступно всем.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body body RegisterRequest true "Логин и пароль"
// @Success      201 {object} UserJSON
// @Failure      400 "неверное тело запроса"
// @Failure      409 "логин занят"
// @Router       /api/users/register [post]
func (a *API) Register(ctx *gin.Context) {
	var body RegisterRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}
	if a.Repo.UsernameExists(body.Username) {
		ctx.Status(http.StatusConflict)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	u := repository.User{Username: body.Username, PasswordHash: string(hash), Role: "creator"}
	if err := a.Repo.CreateUser(&u); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.JSON(http.StatusCreated, UserJSON{ID: u.ID, Username: u.Username})
}

// 9–10. Login и Logout — в auth.go
