package repository

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var catalogIsolation = sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}

type CatalogCost struct {
	BreakEvenPoint
	LikesCount int64
}

func (r *Repository) Catalog(ctx context.Context, from, to *float64, fixed *bool, page, size int) ([]CatalogCost, int64, error) {
	var costs []CatalogCost
	var total int64
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&BreakEvenPoint{}).Where("status = 'published'")
		if from != nil {
			query = query.Where("amount_monthly >= ?", *from)
		}
		if to != nil {
			query = query.Where("amount_monthly <= ?", *to)
		}
		if fixed != nil {
			query = query.Where("is_fixed = ?", *fixed)
		}
		if err := query.Count(&total).Error; err != nil {
			return err
		}
		return query.Select("breakeven_point.*, (SELECT count(*) FROM likes WHERE likes.cost_id = breakeven_point.id) AS likes_count").
			Order("amount_monthly ASC, id ASC").Limit(size).Offset((page - 1) * size).Find(&costs).Error
	}, &catalogIsolation)
	return costs, total, err
}

func (r *Repository) RandomCostIDs(ctx context.Context, previous []int) ([]int, error) {
	var ids []int
	query := r.DB.WithContext(ctx).Model(&BreakEvenPoint{}).Where("status = 'published'")
	if len(previous) > 0 {
		err := query.Clauses(clause.OrderBy{Expression: gorm.Expr("(id IN ?) ASC, random()", previous)}).Limit(100).Pluck("id", &ids).Error
		return ids, err
	}
	err := query.Order("random()").Limit(100).Pluck("id", &ids).Error
	return ids, err
}
