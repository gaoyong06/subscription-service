package biz

import "context"

// Plan 订阅套餐
type Plan struct {
	ID           string
	Name         string
	Description  string
	Price        float64
	Currency     string
	DurationDays int
	Type         string
}

// PlanRepo 套餐仓库接口
type PlanRepo interface {
	ListPlans(ctx context.Context) ([]*Plan, error)
	GetPlan(ctx context.Context, id string) (*Plan, error)
}

// ListPlans 获取所有订阅套餐列表
func (uc *SubscriptionUsecase) ListPlans(ctx context.Context) ([]*Plan, error) {
	return uc.planRepo.ListPlans(ctx)
}

// GetPlan 获取套餐信息
func (uc *SubscriptionUsecase) GetPlan(ctx context.Context, planID string) (*Plan, error) {
	return uc.planRepo.GetPlan(ctx, planID)
}
