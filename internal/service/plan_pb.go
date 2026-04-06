package service

import (
	"context"

	pb "subscription-service/api/subscription/v1"
	"subscription-service/internal/biz"
)

// bizPlanToPB 构造套餐 PB（含区域 pricing，供 ListPlans 一次返回）
func (s *SubscriptionService) bizPlanToPB(ctx context.Context, p *biz.Plan) (*pb.Plan, error) {
	pricings, err := s.uc.ListPlanPricings(ctx, p.PlanID)
	if err != nil {
		return nil, err
	}
	out := &pb.Plan{
		PlanId:        p.PlanID,
		AppId:         p.AppID,
		Name:          p.Name,
		Description:   p.Description,
		Price:         p.Price,
		Currency:      p.Currency,
		Type:          p.Type,
		PeriodType:    biz.NormalizePeriodType(p.PeriodType),
		IntervalCount: p.IntervalCount,
		Features:      append([]string(nil), p.Features...),
	}
	out.Pricings = make([]*pb.PlanPricing, len(pricings))
	for i, x := range pricings {
		out.Pricings[i] = &pb.PlanPricing{
			PlanPricingId: x.PlanPricingID,
			PlanId:        x.PlanID,
			CountryCode:   x.CountryCode,
			Price:         x.Price,
			Currency:      x.Currency,
		}
	}
	return out, nil
}
