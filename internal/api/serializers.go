package api

import "time"


type CostJSON struct {
	ID            int        `json:"id"`
	CostName      string     `json:"cost_name"`
	ShortDesc     string     `json:"short_description"`
	FullDesc      string     `json:"full_description"`
	ImageKey      string     `json:"image_key"`
	VideoKey      string     `json:"video_key"`
	AmountMonthly *float64   `json:"amount_monthly"`
	CostKind      *string    `json:"cost_kind"`
	CreatorID     int        `json:"creator_id"`
	CreatedAt     time.Time  `json:"created_at"`
	FormedAt      *time.Time `json:"formed_at"`
	ImageURL      string     `json:"image_url"`
	VideoURL      string     `json:"video_url"`
	LikesCount    int64      `json:"likes_count"`
	IsMine        bool       `json:"is_mine"`
	IsLiked       bool       `json:"is_liked"`
}

type UserJSON struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type LikeRequest struct {
	Like int `json:"like"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type PublishRequest struct {
	ShortDescription string   `json:"short_description"`
	FullDescription  string   `json:"full_description"`
	AmountMonthly    *float64 `json:"amount_monthly"`
	CostKind         *string  `json:"cost_kind"`
}