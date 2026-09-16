package repository

import (
	"fmt"
	"strconv"
	"strings"
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

func (c CostType) KindLabel() string {
	if c.CostKind == "fixed" {
		return "Постоянные"
	}
	return "Переменные"
}

func (c CostType) AmountText() string { return formatMoney(int(c.AmountMonthly)) + " ₽/мес" }

type RequestLink struct {
	CostID  int
	Volume  int
	Comment string
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
			ID: 1, Status: "черновик", ProductName: "Новая продуктовая линейка",
			SellingPrice: 1250, BepUnits: 8350, BepRevenue: 10437500,
			Links: []RequestLink{
				{CostID: 1, Volume: 1, Comment: "Офис, контракт на 11 мес."},
				{CostID: 2, Volume: 1, Comment: "Управленческая команда"},
				{CostID: 3, Volume: 1, Comment: "Норматив партии"},
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
		if fixed || variable {
			if c.CostKind == "fixed" && !fixed {
				continue
			}
			if c.CostKind == "variable" && !variable {
				continue
			}
		}
		res = append(res, c)
	}
	return res
}

func (r *Repository) GetCostTypeByID(id int) (CostType, error) {
	for _, c := range r.CostTypes {
		if c.ID == id {
			return c, nil
		}
	}
	return CostType{}, fmt.Errorf("вид затрат не найден")
}

func (r *Repository) GetNextCostID(id int) int { return id%len(r.CostTypes) + 1 }

func (r *Repository) GetRequestPositions() int { return len(r.Request.Links) }

// ToggleLike — лайк: сердечко загорается и счётчик растёт (повторно — гаснет).
func (r *Repository) ToggleLike(id int) {
	for i := range r.CostTypes {
		if r.CostTypes[i].ID == id {
			if r.CostTypes[i].Liked {
				r.CostTypes[i].Liked = false
				r.CostTypes[i].Likes--
			} else {
				r.CostTypes[i].Liked = true
				r.CostTypes[i].Likes++
			}
			return
		}
	}
}

// AddCostType — «Сохранить заявку» создаёт новую карточку в «Видах затрат».
func (r *Repository) AddCostType(ct CostType) CostType {
	ct.ID = len(r.CostTypes) + 1
	ct.IsActive = true
	ct.Likes = 0
	r.CostTypes = append(r.CostTypes, ct)
	return ct
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