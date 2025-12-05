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
func (r *planRepo) ListPlans(ctx context.Context, appID string) ([]*biz.Plan, error) {
	var models []model.Plan
	query := r.data.db.WithContext(ctx)
	if appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if err := query.Find(&models).Error; err != nil {
		r.log.Errorf("Failed to list plans: %v", err)
		return nil, err
	}

	plans := make([]*biz.Plan, len(models))
	for i, m := range models {
		plans[i] = &biz.Plan{
			ID:           m.ID,
			AppID:        m.AppID,
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
		AppID:        m.AppID,
		Name:         m.Name,
		Description:  m.Description,
		Price:        m.Price,
		Currency:     m.Currency,
		DurationDays: m.DurationDays,
		Type:         m.Type,
	}, nil
}

// CreatePlan 创建套餐
func (r *planRepo) CreatePlan(ctx context.Context, plan *biz.Plan) error {
	m := &model.Plan{
		ID:           plan.ID,
		AppID:        plan.AppID,
		Name:         plan.Name,
		Description:  plan.Description,
		Price:        plan.Price,
		Currency:     plan.Currency,
		DurationDays: plan.DurationDays,
		Type:         plan.Type,
	}
	if err := r.data.db.WithContext(ctx).Create(m).Error; err != nil {
		r.log.Errorf("Failed to create plan: %v", err)
		return err
	}
	return nil
}

// UpdatePlan 更新套餐
func (r *planRepo) UpdatePlan(ctx context.Context, plan *biz.Plan) error {
	m := &model.Plan{
		ID:           plan.ID,
		AppID:        plan.AppID,
		Name:         plan.Name,
		Description:  plan.Description,
		Price:        plan.Price,
		Currency:     plan.Currency,
		DurationDays: plan.DurationDays,
		Type:         plan.Type,
	}
	if err := r.data.db.WithContext(ctx).Model(&model.Plan{}).Where("plan_id = ?", plan.ID).Updates(m).Error; err != nil {
		r.log.Errorf("Failed to update plan: %v", err)
		return err
	}
	return nil
}

// DeletePlan 删除套餐
func (r *planRepo) DeletePlan(ctx context.Context, id string) error {
	if err := r.data.db.WithContext(ctx).Delete(&model.Plan{}, "plan_id = ?", id).Error; err != nil {
		r.log.Errorf("Failed to delete plan: %v", err)
		return err
	}
	return nil
}

// GetPlanPricing 根据套餐ID和区域获取定价
func (r *planRepo) GetPlanPricing(ctx context.Context, planID, region string) (*biz.PlanPricing, error) {
	var m model.PlanPricing
	if err := r.data.db.WithContext(ctx).Where("plan_id = ? AND region = ?", planID, region).First(&m).Error; err != nil {
		r.log.Warnf("Failed to get plan pricing for %s in region %s: %v", planID, region, err)
		return nil, err
	}
	return &biz.PlanPricing{
		ID:       m.ID,
		PlanID:   m.PlanID,
		Region:   m.Region,
		Price:    m.Price,
		Currency: m.Currency,
	}, nil
}

// ListPlanPricings 获取套餐的所有区域定价
func (r *planRepo) ListPlanPricings(ctx context.Context, planID string) ([]*biz.PlanPricing, error) {
	var models []model.PlanPricing
	if err := r.data.db.WithContext(ctx).Where("plan_id = ?", planID).Find(&models).Error; err != nil {
		r.log.Errorf("Failed to list plan pricings for %s: %v", planID, err)
		return nil, err
	}

	pricings := make([]*biz.PlanPricing, len(models))
	for i, m := range models {
		pricings[i] = &biz.PlanPricing{
			ID:       m.ID,
			PlanID:   m.PlanID,
			Region:   m.Region,
			Price:    m.Price,
			Currency: m.Currency,
		}
	}
	return pricings, nil
}
