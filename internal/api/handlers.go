package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// ===== Функция-singleton: пользователь-создатель зафиксирован константой (до ЛР-4) =====

const creatorID = 1

var (
	currentUserOnce sync.Once
	currentUser     int
)

// CurrentUserID — singleton: значение вычисляется один раз и используется во всех методах
func CurrentUserID() int {
	currentUserOnce.Do(func() { currentUser = creatorID })
	return currentUser
}

type API struct {
	Repo *repository.Repository
	MC   *minio.Client
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

func (a *API) toCostJSON(c repository.BreakEvenType) BreakEvenTypeJSON {
	uid := CurrentUserID()
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
		IsMine:           boolTo01(c.CreatorID == uid),
	}
}

// ===================== ДОМЕН «УСЛУГА» (/api/breakeven_point) =====================

// 1. GET /api/breakeven_point?min_amount=&max_amount=&is_fixed=0|1 — список опубликованных, фильтр на бэкенде
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
		res = append(res, a.toCostJSON(c))
	}
	ctx.JSON(http.StatusOK, res)
}

//  2. GET /api/breakeven_point/feed            — первая опубликованная (без id)
//     GET /api/breakeven_point/feed/:id        — опубликованная по id
//     GET /api/breakeven_point/feed/:id?next=1 — следующая после id (next=true тоже принимается)
//
// Из БД всегда читается одна строка; в ответе like=0/1 для текущего пользователя.
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
	ctx.JSON(http.StatusOK, a.toCostJSON(c))
}

// 3. GET /api/breakeven_point/draft — черновик текущего пользователя (не более одного), id не передаётся
func (a *API) GetDraft(ctx *gin.Context) {
	draft, err := a.Repo.GetDraftByUser(CurrentUserID())
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	ctx.JSON(http.StatusOK, a.toCostJSON(*draft))
}

// 4. POST /api/breakeven_point — создание черновика: name + файлы image и video (multipart/form-data).
// Файлы кладутся в MinIO под латинскими именами, имена сохраняются в image_url/video_url.
func (a *API) CreateCost(ctx *gin.Context) {
	uid := CurrentUserID()
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
	ctx.JSON(http.StatusCreated, a.toCostJSON(*c))
}

// 5. PUT /api/breakeven_point/:id/publish — draft → published (только свой черновик; обратно нельзя)
func (a *API) PublishCost(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	c, err := a.Repo.GetCostByID(id)
	if err != nil || c.Status == "deleted" {
		ctx.Status(http.StatusNotFound)
		return
	}
	if c.CreatorID != CurrentUserID() || c.Status != "draft" {
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
	ctx.JSON(http.StatusOK, a.toCostJSON(c))
}

// 6. DELETE /api/breakeven_point/:id — только soft delete своей услуги (status = 'deleted', ORM)
func (a *API) DeleteCost(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	c, err := a.Repo.GetCostByID(id)
	if err != nil || c.Status == "deleted" {
		ctx.Status(http.StatusNotFound)
		return
	}
	if c.CreatorID != CurrentUserID() {
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
	if err := a.Repo.SetLike(CurrentUserID(), id, *body.Like == 1); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"like": *body.Like, "likes_count": a.Repo.GetLikesCount(id)})
}

// ===================== ДОМЕН «ПОЛЬЗОВАТЕЛЬ» (/api/users) =====================

// 8. POST /api/users/register — {username, password} → 201 {id, username}; 409 если логин занят
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
	h := sha256.Sum256([]byte(body.Password))
	u := repository.User{Username: body.Username, PasswordHash: hex.EncodeToString(h[:]), Role: "creator"}
	if err := a.Repo.CreateUser(&u); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.JSON(http.StatusCreated, UserJSON{ID: u.ID, Username: u.Username})
}

// 9. POST /api/users/login — заглушка аутентификации (реализуется в ЛР-4)
func (a *API) Login(ctx *gin.Context) { ctx.Status(http.StatusOK) }

// 10. POST /api/users/logout — заглушка деавторизации (реализуется в ЛР-4)
func (a *API) Logout(ctx *gin.Context) { ctx.Status(http.StatusOK) }
