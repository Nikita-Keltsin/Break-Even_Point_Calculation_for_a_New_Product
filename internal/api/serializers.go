package api

import "time"

// ===== СЕРИАЛИЗАТОРЫ (то, что уходит клиенту и приходит от клиента) =====

// BreakEvenPointJSON — общие поля вида затрат (таблица breakeven_point) в ответах API.
// Статус на клиент не передаётся; удалённые записи не отдаются вовсе.
type BreakEvenPointJSON struct {
	ID               int        `json:"id"`
	CostName         string     `json:"cost_name"`
	ShortDescription *string    `json:"short_description"` // null у черновика
	ImageURL         string     `json:"image_url"`         // "" → клиент показывает фото по умолчанию
	VideoURL         string     `json:"video_url"`         // "" → клиент показывает видео по умолчанию
	AmountMonthly    *float64   `json:"amount_monthly"`    // null у черновика
	IsFixed          *bool      `json:"is_fixed"`          // null у черновика
	CreatorID        int        `json:"creator_id"`
	CreatedAt        time.Time  `json:"created_at"`
	FormedAt         *time.Time `json:"formed_at"` // null у черновика
	LikesCount       int64      `json:"likes_count"`
}

// BreakEvenPointListJSON — карточка СПИСКА (плитка «Виды затрат»):
// is_mine = 0/1 — карточка текущего пользователя (по нему показывается кнопка «Удалить»)
type BreakEvenPointListJSON struct {
	BreakEvenPointJSON
	IsMine int `json:"is_mine"`
}

// BreakEvenPointFeedJSON — карточка ЛЕНТЫ:
// is_liked = 0/1 — поставил ли текущий пользователь лайк (по нему закрашено сердечко
// и клиент решает, что отправить в POST /like: 1 или 0)
type BreakEvenPointFeedJSON struct {
	BreakEvenPointJSON
	IsLiked int `json:"is_liked"`
}

// LikeResponse — ответ POST /api/breakeven_point/:id/like
type LikeResponse struct {
	IsLiked    int   `json:"is_liked"`
	LikesCount int64 `json:"likes_count"`
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
