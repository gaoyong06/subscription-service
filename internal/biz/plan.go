package biz

import "context"

// Plan 订阅套餐
type Plan struct {
	PlanID       string
	AppID        string // 应用ID（关联api-key-service的app表）
	UID          string // 开发者ID（用户ID，关联api-key-service的app.uid）
	Name         string
	Description  string
	Price        float64 // 默认价格（用于兜底，如果plan_pricing表中没有对应地域的价格）
	Currency     string  // 默认币种（用于兜底）
	DurationDays int
	Type         string
}

// PlanPricing 套餐区域定价（所有价格都在数据库中配置）
type PlanPricing struct {
	PlanPricingID uint64
	PlanID        string
	AppID         string // 应用ID（冗余字段，便于按app查询）
	CountryCode   string // ISO 3166-1 alpha-2 国家代码（如CN, US, DE等）
	Price         float64
	Currency      string
}

// PlanRepo 套餐仓库接口
type PlanRepo interface {
	ListPlans(ctx context.Context, appID string) ([]*Plan, error)
	GetPlan(ctx context.Context, id string) (*Plan, error)
	CreatePlan(ctx context.Context, plan *Plan) error
	UpdatePlan(ctx context.Context, plan *Plan) error
	DeletePlan(ctx context.Context, id string) error
	GetPlanPricing(ctx context.Context, planID, countryCode string) (*PlanPricing, error)
	ListPlanPricings(ctx context.Context, planID string) ([]*PlanPricing, error)
	GetPlanPricingByID(ctx context.Context, planPricingID uint64) (*PlanPricing, error)
	CreatePlanPricing(ctx context.Context, pricing *PlanPricing) error
	UpdatePlanPricing(ctx context.Context, planPricingID uint64, price float64, currency string) error
	DeletePlanPricing(ctx context.Context, planPricingID uint64) error
}

// ListPlans 获取所有订阅套餐列表
func (uc *SubscriptionUsecase) ListPlans(ctx context.Context, appID string) ([]*Plan, error) {
	return uc.planRepo.ListPlans(ctx, appID)
}

// CreatePlan 创建套餐
func (uc *SubscriptionUsecase) CreatePlan(ctx context.Context, plan *Plan) error {
	return uc.planRepo.CreatePlan(ctx, plan)
}

// UpdatePlan 更新套餐
func (uc *SubscriptionUsecase) UpdatePlan(ctx context.Context, plan *Plan) error {
	return uc.planRepo.UpdatePlan(ctx, plan)
}

// DeletePlan 删除套餐
func (uc *SubscriptionUsecase) DeletePlan(ctx context.Context, id string) error {
	return uc.planRepo.DeletePlan(ctx, id)
}

// GetPlan 获取套餐信息
func (uc *SubscriptionUsecase) GetPlan(ctx context.Context, planID string) (*Plan, error) {
	return uc.planRepo.GetPlan(ctx, planID)
}

// GetPlanPricing 获取套餐区域定价（根据国家代码）
// 如果找不到对应国家代码的价格，返回plan表中的默认价格
func (uc *SubscriptionUsecase) GetPlanPricing(ctx context.Context, planID, countryCode string) (*PlanPricing, error) {
	pricing, err := uc.planRepo.GetPlanPricing(ctx, planID, countryCode)
	if err != nil || pricing == nil {
		// 如果没有找到区域定价，返回默认价格
		plan, err := uc.planRepo.GetPlan(ctx, planID)
		if err != nil {
			return nil, err
		}
		return &PlanPricing{
			PlanID:      plan.PlanID,
			AppID:       plan.AppID,
			CountryCode: countryCode,
			Price:       plan.Price,
			Currency:    plan.Currency,
		}, nil
	}
	return pricing, nil
}

// ListPlanPricings 获取套餐的所有区域定价
func (uc *SubscriptionUsecase) ListPlanPricings(ctx context.Context, planID string) ([]*PlanPricing, error) {
	return uc.planRepo.ListPlanPricings(ctx, planID)
}

// GetPlanPricingByID 根据 ID 获取区域定价
func (uc *SubscriptionUsecase) GetPlanPricingByID(ctx context.Context, planPricingID uint64) (*PlanPricing, error) {
	return uc.planRepo.GetPlanPricingByID(ctx, planPricingID)
}

// CreatePlanPricing 创建区域定价
func (uc *SubscriptionUsecase) CreatePlanPricing(ctx context.Context, pricing *PlanPricing) error {
	return uc.planRepo.CreatePlanPricing(ctx, pricing)
}

// UpdatePlanPricing 更新区域定价
func (uc *SubscriptionUsecase) UpdatePlanPricing(ctx context.Context, planPricingID uint64, price float64, currency string) error {
	return uc.planRepo.UpdatePlanPricing(ctx, planPricingID, price, currency)
}

// DeletePlanPricing 删除区域定价
func (uc *SubscriptionUsecase) DeletePlanPricing(ctx context.Context, planPricingID uint64) error {
	return uc.planRepo.DeletePlanPricing(ctx, planPricingID)
}
