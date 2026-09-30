package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Break-Even_Point_Calculation_for_a_New_Product/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// Функция-singleton: пользователь-создатель зафиксирован константой
const currentUserID = 1

func CurrentUserID() uint { return currentUserID }

type API struct {
	Repo *repository.Repository
	MC   *minio.Client
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

func (a *API) toCostJSON(c repository.CostType) CostJSON {
	return CostJSON{
		ID:            c.ID,
		CostName:      c.CostName,
		ShortDesc:     c.ShortDescription,
		FullDesc:      c.FullDescription,
		ImageKey:      c.ImageKey,
		VideoKey:      c.VideoKey,
		AmountMonthly: c.AmountMonthly,
		CostKind:      c.CostKind,
		CreatorID:     c.CreatorID,
		CreatedAt:     c.CreatedAt,
		FormedAt:      c.FormedAt,
		ImageURL:      a.fullURL(c.ImageKey),
		VideoURL:      a.fullURL(c.VideoKey),
		LikesCount:    a.Repo.GetLikesCount(c.ID),
		IsMine:        c.CreatorID == currentUserID,
		IsLiked:       a.Repo.HasLiked(currentUserID, c.ID),
	}
}

// 1. GET /api/costs — список опубликованных, фильтрация на бэкенде
func (a *API) ListCosts(ctx *gin.Context) {
	from, err1 := strconv.ParseFloat(ctx.Query("min_amount"), 64)
	to, err2 := strconv.ParseFloat(ctx.Query("max_amount"), 64)
	hasFrom := err1 == nil && ctx.Query("min_amount") != ""
	hasTo := err2 == nil && ctx.Query("max_amount") != ""
	kind := ctx.Query("kind")
	costs, err := a.Repo.GetPublishedCosts(from, to, hasFrom, hasTo, kind == "fixed", kind == "variable")
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	res := make([]CostJSON, 0, len(costs))
	for _, c := range costs {
		res = append(res, a.toCostJSON(c))
	}
	ctx.JSON(http.StatusOK, res)
}

// 2. GET /api/costs/feed и /api/costs/feed/:id?next=true
func (a *API) FeedCost(ctx *gin.Context) {
	idStr := ctx.Param("id")
	if idStr == "" {
		c, err := a.Repo.GetFeedCost(0)
		if err != nil {
			ctx.Status(http.StatusNotFound)
			return
		}
		ctx.JSON(http.StatusOK, a.toCostJSON(c))
		return
	}
	id, _ := strconv.Atoi(idStr)
	if ctx.Query("next") == "true" {
		id = a.Repo.GetNextPublishedID(id) // в конце ленты заворот на первую
	}
	c, err := a.Repo.GetFeedCost(id)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	ctx.JSON(http.StatusOK, a.toCostJSON(c))
}

// 3. GET /api/costs/draft — черновик текущего пользователя, id с клиента не передаётся
func (a *API) GetDraft(ctx *gin.Context) {
	draft, err := a.Repo.GetDraftByUser(currentUserID)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	ctx.JSON(http.StatusOK, a.toCostJSON(*draft))
}

// 4. POST /api/costs — черновик + файлы image/video в MinIO (multipart)
func (a *API) CreateCost(ctx *gin.Context) {
	if _, err := a.Repo.GetDraftByUser(currentUserID); err == nil {
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
		if key, uerr := UploadFile(a.MC, currentUserID, f, h.Filename, h.Header.Get("Content-Type")); uerr == nil {
			imageKey = key
		}
	}
	if f, h, err := ctx.Request.FormFile("video"); err == nil {
		defer f.Close()
		if key, uerr := UploadFile(a.MC, currentUserID, f, h.Filename, h.Header.Get("Content-Type")); uerr == nil {
			videoKey = key
		}
	}
	c, err := a.Repo.CreateDraftWithMedia(currentUserID, name, imageKey, videoKey)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.JSON(http.StatusCreated, a.toCostJSON(*c))
}

// 5. PUT /api/costs/:id/publish — только draft→published, только свой; назад в черновик нельзя
func (a *API) PublishCost(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	c, err := a.Repo.GetCostByID(id)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	if c.CreatorID != currentUserID || c.Status != "draft" {
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
		"full_description":  body.FullDescription,
		"status":            "published",
		"formed_at":         time.Now(),
	}
	if body.AmountMonthly != nil {
		updates["amount_monthly"] = *body.AmountMonthly
	}
	if body.CostKind != nil {
		updates["cost_kind"] = *body.CostKind
	}
	if err := a.Repo.UpdateCost(id, updates); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	c, _ = a.Repo.GetCostByID(id)
	ctx.JSON(http.StatusOK, a.toCostJSON(c))
}

// 6. DELETE /api/costs/:id — только soft delete, только своя услуга
func (a *API) DeleteCost(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	c, err := a.Repo.GetCostByID(id)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	if c.CreatorID != currentUserID {
		ctx.Status(http.StatusForbidden)
		return
	}
	if err := a.Repo.DeleteCostSQL(id); err != nil { // SQL UPDATE через курсор, без ORM
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.Status(http.StatusOK)
}

// 7. POST /api/costs/:id/like — {"like":1} ставит, {"like":0} отменяет
func (a *API) LikeCost(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	if _, err := a.Repo.GetCostByID(id); err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	var body LikeRequest
	if err := ctx.ShouldBindJSON(&body); err != nil || (body.Like != 0 && body.Like != 1) {
		ctx.Status(http.StatusBadRequest)
		return
	}
	if err := a.Repo.SetLike(currentUserID, id, body.Like == 1); err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.Status(http.StatusOK)
}

// 8. POST /api/users/register — 201 + {id, username}; 409 если логин занят
func (a *API) Register(ctx *gin.Context) {
	var body RegisterRequest
	if err := ctx.ShouldBindJSON(&body); err != nil || body.Username == "" || body.Password == "" {
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

// 9-10. Заглушки для ЛР4
func (a *API) Login(ctx *gin.Context)  { ctx.JSON(http.StatusOK, gin.H{}) }
func (a *API) Logout(ctx *gin.Context) { ctx.JSON(http.StatusOK, gin.H{}) }