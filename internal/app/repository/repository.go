package repository

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Фото и видео по умолчанию лежат на SSR-сервере (resources/media), а не в MinIO
const (
	DefaultImageURL = "/static/media/default-image.png"
	DefaultVideoURL = "/static/media/default-video.mp4"
)

// ===== МОДЕЛИ (таблицы users, breakeven_types, likes) =====

type User struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"type:varchar(50);uniqueIndex;not null"`
	PasswordHash string `gorm:"column:password_hash;type:varchar(100);not null"`
	Email        string `gorm:"type:varchar(100)"`
	Role         string `gorm:"type:varchar(20);not null;default:'creator'"`
}

// BreakEvenType — услуга (вид затрат для расчёта точки безубыточности), таблица breakeven_types
type BreakEvenType struct {
	ID               int       `gorm:"primaryKey"`
	CostName         string    `gorm:"type:varchar(100);not null"`
	ShortDescription string    `gorm:"type:varchar(255)"`
	Status           string    `gorm:"type:varchar(20);not null;default:'draft'"`
	ImageURL         string    `gorm:"column:image_url;type:varchar(255);not null;default:''"`
	VideoURL         string    `gorm:"column:video_url;type:varchar(255);not null;default:''"`
	AmountMonthly    *float64  `gorm:"type:numeric(12,2)"`     // поле по теме 1
	IsFixed          bool      `gorm:"not null;default:false"` // поле по теме 2
	CreatedAt        time.Time `gorm:"not null;default:now()"`
	CreatorID        int       `gorm:"not null"`
	FormedAt         *time.Time
}

// TableName — имя таблицы в БД (по умолчанию GORM назвал бы break_even_types)
func (BreakEvenType) TableName() string { return "breakeven_types" }

type Like struct {
	ID     uint `gorm:"primaryKey"`
	UserID int  `gorm:"not null"`
	CostID int  `gorm:"not null"`
}

// ----- помощники для шаблонов -----

func (c BreakEvenType) KindLabel() string {
	if c.IsFixed {
		return "Постоянные"
	}
	return "Переменные"
}

func (c BreakEvenType) AmountText() string {
	if c.AmountMonthly == nil {
		return "—"
	}
	return formatMoney(int(*c.AmountMonthly)) + " ₽/мес"
}

func (c BreakEvenType) AmountValue() string {
	if c.AmountMonthly == nil {
		return ""
	}
	return strconv.FormatFloat(*c.AmountMonthly, 'f', -1, 64)
}

type Repository struct{ DB *gorm.DB }

func NewRepository(db *gorm.DB) (*Repository, error) { return &Repository{DB: db}, nil }

// Пустой url в БД → фото/видео по умолчанию с SSR
func (r *Repository) ImageURL(c BreakEvenType) string {
	if c.ImageURL == "" {
		return DefaultImageURL
	}
	return c.ImageURL
}

func (r *Repository) VideoURL(c BreakEvenType) string {
	if c.VideoURL == "" {
		return DefaultVideoURL
	}
	return c.VideoURL
}

// ===== ЛР-2: ORM =====

// GetPublishedCosts — поиск/фильтрация опубликованных услуг (ORM, фильтр в SQL WHERE)
// kind: "fixed" | "variable" | "" (все)
func (r *Repository) GetPublishedCosts(from, to float64, hasFrom, hasTo bool, kind string) ([]BreakEvenType, error) {
	q := r.DB.Where("status = ?", "published")
	if hasFrom {
		q = q.Where("amount_monthly >= ?", from)
	}
	if hasTo {
		q = q.Where("amount_monthly <= ?", to)
	}
	switch kind {
	case "fixed":
		q = q.Where("is_fixed = ?", true)
	case "variable":
		q = q.Where("is_fixed = ?", false)
	}
	var res []BreakEvenType
	return res, q.Order("id ASC").Find(&res).Error
}

// GetFeedCost — ЛЕНТА: из БД возвращается ровно одна строка (First = LIMIT 1)
func (r *Repository) GetFeedCost(id int) (BreakEvenType, error) {
	var c BreakEvenType
	q := r.DB.Where("status = ?", "published")
	if id > 0 {
		q = q.Where("id = ?", id)
	}
	return c, q.Order("id ASC").First(&c).Error
}

// GetNextFeedCost — следующая опубликованная после id (одна строка); в конце ленты — первая
func (r *Repository) GetNextFeedCost(id int) (BreakEvenType, error) {
	var c BreakEvenType
	err := r.DB.Where("status = ? AND id > ?", "published", id).Order("id ASC").First(&c).Error
	if err != nil {
		return r.GetFeedCost(0)
	}
	return c, nil
}

func (r *Repository) GetLikesCount(costID int) int64 {
	var n int64
	r.DB.Model(&Like{}).Where("cost_id = ?", costID).Count(&n)
	return n
}

func (r *Repository) HasLiked(uid, costID int) bool {
	var n int64
	r.DB.Model(&Like{}).Where("user_id = ? AND cost_id = ?", uid, costID).Count(&n)
	return n > 0
}

func (r *Repository) GetDraftByUser(uid int) (*BreakEvenType, error) {
	var c BreakEvenType
	err := r.DB.Where("creator_id = ? AND status = ?", uid, "draft").First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateDraft — «Далее»: создаём черновик только с названием (ORM).
// Фото и видео в ЛР-2 не сохраняются → url пустые → показываются файлы по умолчанию.
func (r *Repository) CreateDraft(uid int, name string) error {
	c := BreakEvenType{CostName: name, Status: "draft", CreatorID: uid}
	return r.DB.Create(&c).Error
}

// PublishDraft — «Опубликовать»: краткое описание + 2 поля по теме, смена статуса (ORM)
func (r *Repository) PublishDraft(uid int, description string, amount float64, isFixed bool) error {
	draft, err := r.GetDraftByUser(uid)
	if err != nil {
		return err
	}
	return r.DB.Model(&BreakEvenType{}).Where("id = ?", draft.ID).Updates(map[string]interface{}{
		"short_description": description,
		"amount_monthly":    amount,
		"is_fixed":          isFixed,
		"status":            "published",
		"formed_at":         time.Now(),
	}).Error
}

// DeleteCostSQL — логическое удаление через SQL UPDATE (курсор database/sql, без ORM)
func (r *Repository) DeleteCostSQL(id int) error {
	sqlDB, err := r.DB.DB()
	if err != nil {
		return err
	}
	res, err := sqlDB.Exec("UPDATE breakeven_types SET status = 'deleted' WHERE id = $1 AND status <> 'deleted'", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("услуга не найдена или уже удалена")
	}
	return nil
}

// ===== ЛР-3: ORM для веб-сервиса =====

func (r *Repository) GetCostByID(id int) (BreakEvenType, error) {
	var c BreakEvenType
	return c, r.DB.First(&c, id).Error
}

func (r *Repository) UpdateCost(id int, updates map[string]interface{}) error {
	return r.DB.Model(&BreakEvenType{}).Where("id = ?", id).Updates(updates).Error
}

// SoftDeleteCost — DELETE в API: только смена статуса (soft delete) через ORM
func (r *Repository) SoftDeleteCost(id int) error {
	return r.DB.Model(&BreakEvenType{}).Where("id = ? AND status <> ?", id, "deleted").
		Update("status", "deleted").Error
}

// SetLike — like=1 ставит лайк, like=0 снимает
func (r *Repository) SetLike(uid, costID int, set bool) error {
	if set {
		if r.HasLiked(uid, costID) {
			return nil
		}
		return r.DB.Create(&Like{UserID: uid, CostID: costID}).Error
	}
	return r.DB.Where("user_id = ? AND cost_id = ?", uid, costID).Delete(&Like{}).Error
}

func (r *Repository) CreateDraftWithMedia(uid int, name, imageKey, videoKey string) (*BreakEvenType, error) {
	c := BreakEvenType{CostName: name, Status: "draft", CreatorID: uid, ImageURL: imageKey, VideoURL: videoKey}
	if err := r.DB.Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) UsernameExists(name string) bool {
	var n int64
	r.DB.Model(&User{}).Where("username = ?", name).Count(&n)
	return n > 0
}

func (r *Repository) CreateUser(u *User) error { return r.DB.Create(u).Error }

// ===== ЛР-4: аутентификация =====

// GetUserByUsername — пользователь по логину (для проверки пароля при входе)
func (r *Repository) GetUserByUsername(name string) (User, error) {
	var u User
	return u, r.DB.Where("username = ?", name).First(&u).Error
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
