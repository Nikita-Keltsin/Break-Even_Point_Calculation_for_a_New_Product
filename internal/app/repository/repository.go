package repository

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Дефолты лежат на SSR (resources/media), НЕ в MinIO
const (
	DefaultImageURL = "/static/media/default-image.png"
	DefaultVideoURL = "/static/media/default-video.mp4"
	DefaultImageKey = "screen.png"
	DefaultVideoKey = "7426703-hd_1080_1920_25fps.mp4"
)

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"type:varchar(50);uniqueIndex;not null"`
	Email    string `gorm:"type:varchar(100)"`
	Role     string `gorm:"type:varchar(20);not null;default:'creator'"`
}

type CostType struct {
	ID               int       `gorm:"primaryKey"`
	CostName         string    `gorm:"type:varchar(100);not null"`
	ShortDescription string    `gorm:"type:varchar(255)"`
	FullDescription  string    `gorm:"type:text"`
	Status           string    `gorm:"type:varchar(20);not null;default:'draft'"`
	ImageKey         string    `gorm:"column:image_url;type:varchar(255)"`
	VideoKey         string    `gorm:"column:video_url;type:varchar(255)"`
	AmountMonthly    *float64  `gorm:"type:numeric(12,2)"`
	CostKind         *string   `gorm:"type:varchar(20)"`
	CreatedAt        time.Time `gorm:"not null;default:now()"`
	CreatorID        int        `gorm:"not null"`
	FormedAt         *time.Time
}

func (c CostType) KindLabel() string {
	if c.CostKind == nil {
		return "—"
	}
	if *c.CostKind == "fixed" {
		return "Постоянные"
	}
	return "Переменные"
}

func (c CostType) AmountText() string {
	if c.AmountMonthly == nil {
		return "—"
	}
	return formatMoney(int(*c.AmountMonthly)) + " ₽/мес"
}

func (c CostType) AmountValue() string {
	if c.AmountMonthly == nil {
		return ""
	}
	return strconv.FormatFloat(*c.AmountMonthly, 'f', -1, 64)
}

func (c CostType) KindValue() string {
	if c.CostKind == nil {
		return "fixed"
	}
	return *c.CostKind
}

type Like struct {
	ID     uint `gorm:"primaryKey"`
	UserID int  `gorm:"not null"`
	CostID int  `gorm:"not null"`
}

type Repository struct { DB *gorm.DB }

func NewRepository(db *gorm.DB) (*Repository, error) { return &Repository{DB: db}, nil }

func (r *Repository) ImageURL(c CostType) string {
	if c.ImageKey == "" {
		return DefaultImageURL
	}
	return c.ImageKey
}

func (r *Repository) VideoURL(c CostType) string {
	if c.VideoKey == "" {
		return DefaultVideoURL
	}
	return c.VideoKey
}

// ЛЕНТА: одна строка из БД (First = LIMIT 1)
func (r *Repository) GetFeedCost(id int) (CostType, error) {
	var c CostType
	q := r.DB.Where("status = ?", "published")
	if id > 0 {
		q = q.Where("id = ?", id)
	}
	return c, q.Order("id ASC").First(&c).Error
}

func (r *Repository) GetNextPublishedID(id int) int {
	var next CostType
	if err := r.DB.Where("status = ? AND id > ?", "published", id).Order("id ASC").First(&next).Error; err != nil {
		var first CostType
		if e2 := r.DB.Where("status = ?", "published").Order("id ASC").First(&first).Error; e2 != nil {
			return id
		}
		return first.ID
	}
	return next.ID
}

func (r *Repository) GetLikesCount(costID int) int64 {
	var n int64
	r.DB.Model(&Like{}).Where("cost_id = ?", costID).Count(&n)
	return n
}


func (r *Repository) GetPublishedCosts(from, to float64, hasFrom, hasTo, fixed, variable bool) ([]CostType, error) {
	q := r.DB.Where("status = ?", "published")
	if hasFrom {
		q = q.Where("amount_monthly >= ?", from)
	}
	if hasTo {
		q = q.Where("amount_monthly <= ?", to)
	}
	if fixed && !variable {
		q = q.Where("cost_kind = ?", "fixed")
	}
	if variable && !fixed {
		q = q.Where("cost_kind = ?", "variable")
	}
	var res []CostType
	return res, q.Order("id ASC").Find(&res).Error
}

func (r *Repository) GetDraftByUser(uid int) (*CostType, error) {
	var c CostType
	err := r.DB.Where("creator_id = ? AND status = ?", uid, "draft").First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// «Далее»: черновик со ВСЕМИ полями формы (поля = колонки БД)
func (r *Repository) CreateDraft(uid int, data CostType) error {
	data.Status = "draft"
	data.CreatorID = uid
	return r.DB.Create(&data).Error
}

// «Опубликовать»: публикация через ORM
func (r *Repository) PublishDraft(uid int, short, full string, amount float64, kind string) error {
	draft, err := r.GetDraftByUser(uid)
	if err != nil {
		return err
	}
	return r.DB.Model(&CostType{}).Where("id = ?", draft.ID).Updates(map[string]interface{}{
		"short_description": short,
		"full_description":  full,
		"amount_monthly":    amount,
		"cost_kind":         kind,
		"status":            "published",
		"formed_at":         time.Now(),
	}).Error
}


func (r *Repository) DeleteCostSQL(id int) error {
	sqlDB, err := r.DB.DB()
	if err != nil {
		return err
	}
	res, err := sqlDB.Exec("UPDATE cost_types SET status = 'deleted' WHERE id = $1 AND status <> 'deleted'", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("услуга не найдена или уже удалена")
	}
	return nil
}

func formatMoney(n int) string {
	s := strconv.Itoa(n)
	var sb strings.Builder
	for i, cnt := len(s)-1, 0; i >= 0; i-- {
		sb.WriteByte(s[i])
		cnt++
		if cnt%3 == 0 && i != 0 {
			sb.WriteByte(' ')
		}
	}
	b := []byte(sb.String())
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}


func (r *Repository) HasLiked(uid, costID int) bool {
	var n int64
	r.DB.Model(&Like{}).Where("user_id = ? AND cost_id = ?", uid, costID).Count(&n)
	return n > 0
}


func (r *Repository) ToggleLike(uid, costID int) error {
	var like Like
	err := r.DB.Where("user_id = ? AND cost_id = ?", uid, costID).First(&like).Error
	if err == nil {
		return r.DB.Delete(&like).Error
	}
	return r.DB.Create(&Like{UserID: uid, CostID: costID}).Error
}