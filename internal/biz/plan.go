package biz

import "context"

// Plan 订阅套餐
type Plan struct {
	ID           string
	Name         string
	Description  string
	Price        float64 // 默认价格
	Currency     string  // 默认货币
	DurationDays int
	Type         string
}

// PlanPricing 套餐区域定价
type PlanPricing struct {
	ID       uint64
	PlanID   string
	Region   string // US, CN, EU, etc.
	Price    float64
	Currency string
}

// PlanRepo 套餐仓库接口
type PlanRepo interface {
	ListPlans(ctx context.Context) ([]*Plan, error)
	GetPlan(ctx context.Context, id string) (*Plan, error)
	GetPlanPricing(ctx context.Context, planID, region string) (*PlanPricing, error)
	ListPlanPricings(ctx context.Context, planID string) ([]*PlanPricing, error)
}

// ListPlans 获取所有订阅套餐列表
func (uc *SubscriptionUsecase) ListPlans(ctx context.Context) ([]*Plan, error) {
	return uc.planRepo.ListPlans(ctx)
}

// GetPlan 获取套餐信息
func (uc *SubscriptionUsecase) GetPlan(ctx context.Context, planID string) (*Plan, error) {
	return uc.planRepo.GetPlan(ctx, planID)
}

// GetPlanPricing 获取套餐区域定价
func (uc *SubscriptionUsecase) GetPlanPricing(ctx context.Context, planID, region string) (*PlanPricing, error) {
	pricing, err := uc.planRepo.GetPlanPricing(ctx, planID, region)
	if err != nil || pricing == nil {
		// 如果没有找到区域定价，返回默认价格
		plan, err := uc.planRepo.GetPlan(ctx, planID)
		if err != nil {
			return nil, err
		}
		return &PlanPricing{
			PlanID:   plan.ID,
			Region:   "default",
			Price:    plan.Price,
			Currency: plan.Currency,
		}, nil
	}
	return pricing, nil
}
