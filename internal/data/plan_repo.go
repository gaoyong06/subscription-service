package data

import (
	"context"
	"errors"
	"subscription-service/internal/biz"
	"subscription-service/internal/data/model"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
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
			PlanID:        m.PlanID,
			AppID:         m.AppID,
			UserID:        m.UserID,
			Name:          m.Name,
			Description:   m.Description,
			Price:         m.Price,
			Currency:      m.Currency,
			PeriodType:    m.PeriodType,
			IntervalCount: m.IntervalCount,
			Features:      decodePlanFeaturesJSON(m.Features),
			Type:          m.Type,
		}
	}
	return plans, nil
}

// FindFreeForeverPlanByApp 查找应用下 type=free 且 period_type=FOREVER 的套餐（软删排除）
func (r *planRepo) FindFreeForeverPlanByApp(ctx context.Context, appID string) (*biz.Plan, error) {
	if appID == "" {
		return nil, nil
	}
	var m model.Plan
	err := r.data.db.WithContext(ctx).
		Where("app_id = ? AND LOWER(TRIM(type)) = ? AND UPPER(TRIM(period_type)) = ?", appID, "free", "FOREVER").
		Order("plan_id ASC").
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		r.log.Errorf("FindFreeForeverPlanByApp failed: %v", err)
		return nil, err
	}
	return &biz.Plan{
		PlanID:        m.PlanID,
		AppID:         m.AppID,
		UserID:        m.UserID,
		Name:          m.Name,
		Description:   m.Description,
		Price:         m.Price,
		Currency:      m.Currency,
		PeriodType:    m.PeriodType,
		IntervalCount: m.IntervalCount,
		Features:      decodePlanFeaturesJSON(m.Features),
		Type:          m.Type,
	}, nil
}

// GetPlan 根据ID获取套餐
func (r *planRepo) GetPlan(ctx context.Context, id string) (*biz.Plan, error) {
	var m model.Plan
	if err := r.data.db.WithContext(ctx).First(&m, "plan_id = ?", id).Error; err != nil {
		r.log.Errorf("Failed to get plan %s: %v", id, err)
		return nil, err
	}
	return &biz.Plan{
		PlanID:        m.PlanID,
		AppID:         m.AppID,
		UserID:        m.UserID,
		Name:          m.Name,
		Description:   m.Description,
		Price:         m.Price,
		Currency:      m.Currency,
		PeriodType:    m.PeriodType,
		IntervalCount: m.IntervalCount,
		Features:      decodePlanFeaturesJSON(m.Features),
		Type:          m.Type,
	}, nil
}

// CreatePlan 创建套餐
func (r *planRepo) CreatePlan(ctx context.Context, plan *biz.Plan) error {
	m := &model.Plan{
		PlanID:        plan.PlanID,
		AppID:         plan.AppID,
		UserID:        plan.UserID,
		Name:          plan.Name,
		Description:   plan.Description,
		Price:         plan.Price,
		Currency:      plan.Currency,
		PeriodType:    plan.PeriodType,
		IntervalCount: plan.IntervalCount,
		Features:      encodePlanFeaturesJSON(plan.Features),
		Type:          plan.Type,
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
		PlanID:        plan.PlanID,
		AppID:         plan.AppID,
		UserID:        plan.UserID,
		Name:          plan.Name,
		Description:   plan.Description,
		Price:         plan.Price,
		Currency:      plan.Currency,
		PeriodType:    plan.PeriodType,
		IntervalCount: plan.IntervalCount,
		Features:      encodePlanFeaturesJSON(plan.Features),
		Type:          plan.Type,
	}
	if err := r.data.db.WithContext(ctx).Model(&model.Plan{}).Where("plan_id = ?", plan.PlanID).Updates(m).Error; err != nil {
		r.log.Errorf("Failed to update plan: %v", err)
		return err
	}
	return nil
}

// DeletePlan 软删除套餐（设置 deleted_at，不做物理删除）
func (r *planRepo) DeletePlan(ctx context.Context, id string) error {
	res := r.data.db.WithContext(ctx).Where("plan_id = ?", id).Delete(&model.Plan{})
	if res.Error != nil {
		r.log.Errorf("Failed to soft-delete plan: %v", res.Error)
		return res.Error
	}
	if res.RowsAffected == 0 {
		r.log.Warnf("Soft-delete plan: no row updated for plan_id=%s (not found or already deleted)", id)
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetPlanPricing 根据套餐ID和国家代码获取定价
func (r *planRepo) GetPlanPricing(ctx context.Context, planID, countryCode string) (*biz.PlanPricing, error) {
	var m model.PlanPricing
	if err := r.data.db.WithContext(ctx).Where("plan_id = ? AND country_code = ?", planID, countryCode).First(&m).Error; err != nil {
		r.log.Warnf("Failed to get plan pricing for %s in country %s: %v", planID, countryCode, err)
		return nil, err
	}
	return &biz.PlanPricing{
		PlanPricingID: m.PlanPricingID,
		PlanID:        m.PlanID,
		AppID:         m.AppID,
		CountryCode:   m.CountryCode,
		Price:         m.Price,
		Currency:      m.Currency,
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
			PlanPricingID: m.PlanPricingID,
			PlanID:        m.PlanID,
			AppID:         m.AppID,
			CountryCode:   m.CountryCode,
			Price:         m.Price,
			Currency:      m.Currency,
		}
	}
	return pricings, nil
}

// GetPlanPricingByID 根据 ID 获取区域定价
func (r *planRepo) GetPlanPricingByID(ctx context.Context, planPricingID uint64) (*biz.PlanPricing, error) {
	var m model.PlanPricing
	if err := r.data.db.WithContext(ctx).Where("plan_pricing_id = ?", planPricingID).First(&m).Error; err != nil {
		r.log.Errorf("Failed to get plan pricing by ID: %v", err)
		return nil, err
	}
	return &biz.PlanPricing{
		PlanPricingID: m.PlanPricingID,
		PlanID:        m.PlanID,
		AppID:         m.AppID,
		CountryCode:   m.CountryCode,
		Price:         m.Price,
		Currency:      m.Currency,
	}, nil
}

// CreatePlanPricing 创建区域定价
func (r *planRepo) CreatePlanPricing(ctx context.Context, pricing *biz.PlanPricing) error {
	// 如果 AppID 为空，从 plan 表获取
	appID := pricing.AppID
	if appID == "" {
		plan, err := r.GetPlan(ctx, pricing.PlanID)
		if err != nil {
			r.log.Warnf("Failed to get plan for app_id: %v, will use empty string", err)
		} else if plan != nil {
			appID = plan.AppID
		}
	}

	m := &model.PlanPricing{
		PlanID:      pricing.PlanID,
		AppID:       appID,
		CountryCode: pricing.CountryCode,
		Price:       pricing.Price,
		Currency:    pricing.Currency,
	}
	if err := r.data.db.WithContext(ctx).Create(m).Error; err != nil {
		r.log.Errorf("Failed to create plan pricing: %v", err)
		return err
	}
	// 更新返回的 ID 和 AppID
	pricing.PlanPricingID = m.PlanPricingID
	pricing.AppID = m.AppID
	return nil
}

// UpdatePlanPricing 更新区域定价
func (r *planRepo) UpdatePlanPricing(ctx context.Context, planPricingID uint64, price float64, currency string) error {
	if err := r.data.db.WithContext(ctx).Model(&model.PlanPricing{}).
		Where("plan_pricing_id = ?", planPricingID).
		Updates(map[string]interface{}{
			"price":    price,
			"currency": currency,
		}).Error; err != nil {
		r.log.Errorf("Failed to update plan pricing: %v", err)
		return err
	}
	return nil
}

// DeletePlanPricing 删除区域定价
func (r *planRepo) DeletePlanPricing(ctx context.Context, planPricingID uint64) error {
	if err := r.data.db.WithContext(ctx).Delete(&model.PlanPricing{}, "plan_pricing_id = ?", planPricingID).Error; err != nil {
		r.log.Errorf("Failed to delete plan pricing: %v", err)
		return err
	}
	return nil
}
