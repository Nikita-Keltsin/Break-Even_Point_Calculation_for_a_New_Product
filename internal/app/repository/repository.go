package repository

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)


const (
	MediaBaseURL    = "http://localhost:9000/media/"
	DefaultImageKey = "screen.png"
	DefaultVideoKey = "7426703-hd_1080_1920_25fps.mp4"
)

type CostType struct {
	ID               int
	CostName         string
	ShortDescription string
	FullDescription  string
	IsActive         bool
	ImageKey         string
	VideoKey         string
	CostKind         string
	AmountMonthly    float64
	CostShare        int
	Likes            int
	Liked            bool
}

func (c *CostType) normalizeMedia() {
	if c.ImageKey == "" {
		c.ImageKey = MediaBaseURL + DefaultImageKey
	}
	if c.VideoKey == "" {
		c.VideoKey = MediaBaseURL + DefaultVideoKey
	}
}

func (c CostType) KindLabel() string {
	if c.CostKind == "fixed" {
		return "Постоянные"
	}
	return "Переменные"
}

func (c CostType) AmountText() string { return formatMoney(int(c.AmountMonthly)) + " ₽/мес" }

type RequestLink struct {
	CostID     int
	Volume     int
	IsCritical bool
	Comment    string
}

type BreakevenRequest struct {
	ID           int
	Status       string
	ProductName  string
	SellingPrice float64
	BepUnits     int
	BepRevenue   float64
	Links        []RequestLink
}

type Repository struct {
	CostTypes   []CostType
	Request     *BreakevenRequest
	MediaImages []string
	MediaVideos []string
}

func NewRepository() (*Repository, error) {
	return &Repository{
		CostTypes: []CostType{
			{ID: 1, CostName: "Операционные расходы (OPEX)", ShortDescription: "Управление постоянными расходами коммерческой недвижимости и головного офиса для оптимизации маржинальности бизнеса", FullDescription: "Управление постоянными расходами коммерческой недвижимости и головного офиса для оптимизации маржинальности бизнеса. Включает аренду переговорных, коворкинга и представительского офиса, коммунальные платежи, клининг и охрану. Регламент пересмотра ставок — ежеквартально, ответственный — финансовый директор.", IsActive: true, ImageKey: "screen.png", VideoKey: "7426703-hd_1080_1920_25fps.mp4", CostKind: "fixed", AmountMonthly: 2850000, CostShare: 24, Likes: 3},
			{ID: 2, CostName: "Фонд оплаты труда (ФОТ)", ShortDescription: "Постоянные расходы на оклады управленческой команды, головного офиса и линий поддержки", FullDescription: "Постоянные расходы на оклады управленческой команды, головного офиса и линий поддержки. Включает оклады, страховые взносы и ДМС. Не зависит от объёма выручки, индексируется один раз в год по решению комитета по вознаграждениям.", IsActive: true, ImageKey: "screen_2.png", VideoKey: "7652821-hd_1080_1920_25fps.mp4", CostKind: "fixed", AmountMonthly: 4620000, CostShare: 38, Likes: 3},
			{ID: 3, CostName: "Себестоимость продаж (COGS)", ShortDescription: "Переменные затраты производства и логистики, пропорциональные объёму отгрузок", FullDescription: "Переменные затраты производства и логистики, пропорциональные объёму отгрузок: сырьё, упаковка, складская обработка и магистральная доставка до РЦ. Норматив пересчитывается от фактической партии каждую неделю.", IsActive: true, ImageKey: "screen_3.png", VideoKey: "7818630-hd_1080_1920_30fps.mp4", CostKind: "variable", AmountMonthly: 7410000, CostShare: 28, Likes: 3},
			{ID: 4, CostName: "Маркетинг и аналитика", ShortDescription: "Переменные расходы на продвижение, перформанс-кампании и продуктовую аналитику", FullDescription: "Переменные расходы на продвижение, перформанс-кампании и продуктовую аналитику: медиабаинг, сквозная аналитика, A/B-платформы. Планируются от процента выручки, поэтому относятся к переменной части затрат.", IsActive: true, ImageKey: "screen_4.png", VideoKey: "7643847-uhd_2160_4096_25fps.mp4", CostKind: "variable", AmountMonthly: 1150000, CostShare: 10, Likes: 3},
		},
		Request: &BreakevenRequest{
			ID: 1, Status: "черновик", ProductName: "Новая продуктовая линейка", SellingPrice: 1250,
			Links: []RequestLink{
				{CostID: 1, Volume: 1, IsCritical: true, Comment: "Офис, контракт на 11 мес."},
				{CostID: 2, Volume: 1, IsCritical: true, Comment: "Управленческая команда"},
				{CostID: 3, Volume: 1, IsCritical: false, Comment: "Норматив партии"},
			},
		},
		MediaImages: []string{"screen.png", "screen_2.png", "screen_3.png", "screen_4.png"},
		MediaVideos: []string{
			"7426703-hd_1080_1920_25fps.mp4",
			"7652821-hd_1080_1920_25fps.mp4",
			"7818630-hd_1080_1920_30fps.mp4",
			"7643847-uhd_2160_4096_25fps.mp4",
		},
	}, nil
}

func (r *Repository) GetCostTypesFiltered(from, to float64, hasFrom, hasTo, fixed, variable bool) []CostType {
	var res []CostType
	for _, c := range r.CostTypes {
		if !c.IsActive {
			continue
		}
		if hasFrom && c.AmountMonthly < from {
			continue
		}
		if hasTo && c.AmountMonthly > to {
			continue
		}
		if fixed && !variable && c.CostKind != "fixed" {
			continue
		}
		if variable && !fixed && c.CostKind != "variable" {
			continue
		}
		c.normalizeMedia()
		res = append(res, c)
	}
	return res
}

func (r *Repository) GetCostTypeByID(id int) (CostType, error) {
	for _, c := range r.CostTypes {
		if c.ID == id {
			c.normalizeMedia()
			return c, nil
		}
	}
	return CostType{}, fmt.Errorf("вид затрат не найден")
}

func (r *Repository) GetNextCostID(id int) int { return id%len(r.CostTypes) + 1 }

func (r *Repository) GetRequestPositions() int { return len(r.Request.Links) }

func (r *Repository) ToggleLike(id int) {
	for i := range r.CostTypes {
		if r.CostTypes[i].ID == id {
			r.CostTypes[i].Liked = !r.CostTypes[i].Liked
			if r.CostTypes[i].Liked {
				r.CostTypes[i].Likes++
			} else {
				r.CostTypes[i].Likes--
			}
			return
		}
	}
}

func (r *Repository) AddCostType(ct CostType) CostType {
	ct.ID = len(r.CostTypes) + 1
	ct.IsActive = true
	ct.Likes = 0
	r.CostTypes = append(r.CostTypes, ct)
	return ct
}

// Заявка по id (для страницы заявки)
func (r *Repository) GetRequestByID(id int) (*BreakevenRequest, error) {
	if r.Request != nil && r.Request.ID == id {
		return r.Request, nil
	}
	return nil, fmt.Errorf("заявка не найдена")
}

// Логическое удаление заявки (статус → deleted)
func (r *Repository) DeleteRequest(id int) error {
	if r.Request == nil || r.Request.ID != id || r.Request.Status != "черновик" {
		return fmt.Errorf("заявка не найдена или не в статусе черновик")
	}
	r.Request.Status = "deleted"
	return nil
}

// Добавление услуги в заявку; если черновика нет (или удалён) — создаётся новый
func (r *Repository) AddCostToRequest(costID, volume int, comment string) error {
	if r.Request == nil || r.Request.Status == "deleted" {
		nextID := 1
		if r.Request != nil {
			nextID = r.Request.ID + 1
		}
		r.Request = &BreakevenRequest{ID: nextID, Status: "черновик", ProductName: "Новая продуктовая линейка", SellingPrice: 1250}
	}
	for i := range r.Request.Links {
		if r.Request.Links[i].CostID == costID {
			r.Request.Links[i].Volume += volume
			return nil
		}
	}
	r.Request.Links = append(r.Request.Links, RequestLink{CostID: costID, Volume: volume, Comment: comment})
	return nil
}

// Правка поля м-м (объём)
func (r *Repository) UpdateLinkVolume(costID, volume int) error {
	if r.Request == nil {
		return fmt.Errorf("нет заявки")
	}
	for i := range r.Request.Links {
		if r.Request.Links[i].CostID == costID {
			r.Request.Links[i].Volume = volume
			return nil
		}
	}
	return fmt.Errorf("связь не найдена")
}

// ФОРМУЛА: ТБУ(шт) = Пост / (Цена − Перем на ед.); ТБУ(₽) = ТБУ(шт) × Цена
func (r *Repository) ComputeBreakEven(req *BreakevenRequest) (int, float64) {
	var fixed, varPerUnit float64
	for _, l := range req.Links {
		cost, err := r.GetCostTypeByID(l.CostID)
		if err != nil {
			continue
		}
		if cost.CostKind == "fixed" {
			fixed += cost.AmountMonthly
			continue
		}
		vol := l.Volume
		if vol <= 0 {
			vol = 1
		}
		varPerUnit += cost.AmountMonthly / float64(vol)
	}
	margin := req.SellingPrice - varPerUnit
	if margin <= 0 || fixed <= 0 {
		return 0, 0
	}
	units := int(math.Ceil(fixed / margin))
	return units, float64(units) * req.SellingPrice
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