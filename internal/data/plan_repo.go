package data

import (
	"context"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/data/model"

	"github.com/go-kratos/kratos/v2/log"
)

// planRepo Plan 仓库实现
type planRepo struct {
	data *Data
	log  *log.Helper
}

// NewPlanRepo 创建 Plan 仓库
func NewPlanRepo(data *Data, logger log.Logger) biz.PlanRepo {
	return &planRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// ListPlans 获取所有套餐列表
func (r *planRepo) ListPlans(ctx context.Context) ([]*biz.Plan, error) {
	var models []model.Plan
	if err := r.data.db.WithContext(ctx).Find(&models).Error; err != nil {
		r.log.Errorf("Failed to list plans: %v", err)
		return nil, err
	}

	plans := make([]*biz.Plan, len(models))
	for i, m := range models {
		plans[i] = &biz.Plan{
			ID:           m.ID,
			Name:         m.Name,
			Description:  m.Description,
			Price:        m.Price,
			Currency:     m.Currency,
			DurationDays: m.DurationDays,
			Type:         m.Type,
		}
	}
	return plans, nil
}

// GetPlan 根据ID获取套餐
func (r *planRepo) GetPlan(ctx context.Context, id string) (*biz.Plan, error) {
	var m model.Plan
	if err := r.data.db.WithContext(ctx).First(&m, "plan_id = ?", id).Error; err != nil {
		r.log.Errorf("Failed to get plan %s: %v", id, err)
		return nil, err
	}
	return &biz.Plan{
		ID:           m.ID,
		Name:         m.Name,
		Description:  m.Description,
		Price:        m.Price,
		Currency:     m.Currency,
		DurationDays: m.DurationDays,
		Type:         m.Type,
	}, nil
}

