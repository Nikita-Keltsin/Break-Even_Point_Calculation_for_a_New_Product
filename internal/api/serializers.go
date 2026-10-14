package api

import "time"

// ===== СЕРИАЛИЗАТОРЫ (то, что уходит клиенту и приходит от клиента) =====

// BreakEvenTypeJSON — услуга (вид затрат, таблица breakeven_types) в ответах API.
// Статус на клиент не передаётся; удалённые записи не отдаются вовсе.
type BreakEvenTypeJSON struct {
	ID               int        `json:"id"`
	CostName         string     `json:"cost_name"`
	ShortDescription string     `json:"short_description"`
	ImageURL         string     `json:"image_url"` // "" → клиент показывает фото по умолчанию
	VideoURL         string     `json:"video_url"` // "" → клиент показывает видео по умолчанию
	AmountMonthly    *float64   `json:"amount_monthly"`
	IsFixed          bool       `json:"is_fixed"`
	CreatorID        int        `json:"creator_id"`
	CreatedAt        time.Time  `json:"created_at"`
	FormedAt         *time.Time `json:"formed_at"`
	LikesCount       int64      `json:"likes_count"`
	Like             int        `json:"like"`    // 0/1 — поставил ли текущий пользователь лайк
	IsMine           int        `json:"is_mine"` // 0/1 — карточка текущего пользователя
}

type UserJSON struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

// LikeRequest — POST /api/breakeven_point/:id/like, like: 1 ставит, 0 отменяет
type LikeRequest struct {
	Like *int `json:"like" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// PublishRequest — PUT /api/breakeven_point/:id/publish: краткое описание + 2 поля по теме.
// Системные поля (id, status, creator_id, даты) с клиента не принимаются.
type PublishRequest struct {
	ShortDescription string   `json:"short_description" binding:"required"`
	AmountMonthly    *float64 `json:"amount_monthly" binding:"required"`
	IsFixed          *bool    `json:"is_fixed" binding:"required"`
}

// LikeResponse — ответ POST /api/breakeven_point/:id/like
type LikeResponse struct {
	Like       int   `json:"like"`
	LikesCount int64 `json:"likes_count"`
}

// LoginRequest — POST /api/users/login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse — JWT для заголовка Authorization: Bearer <access_token>
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
