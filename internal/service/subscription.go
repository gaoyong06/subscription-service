package service

import (
	"context"
	pb "xinyuan_tech/subscription-service/api/subscription/v1"
	"xinyuan_tech/subscription-service/internal/biz"
)

type SubscriptionService struct {
	pb.UnimplementedSubscriptionServer
	uc *biz.SubscriptionUsecase
}

func NewSubscriptionService(uc *biz.SubscriptionUsecase) *SubscriptionService {
	return &SubscriptionService{uc: uc}
}

func (s *SubscriptionService) ListPlans(ctx context.Context, req *pb.ListPlansRequest) (*pb.ListPlansReply, error) {
	plans, err := s.uc.ListPlans(ctx)
	if err != nil {
		return nil, err
	}

	pbPlans := make([]*pb.Plan, len(plans))
	for i, p := range plans {
		pbPlans[i] = &pb.Plan{
			Id:           p.ID,
			Name:         p.Name,
			Description:  p.Description,
			Price:        p.Price,
			Currency:     p.Currency,
			DurationDays: int32(p.DurationDays),
			Type:         p.Type,
		}
	}

	return &pb.ListPlansReply{Plans: pbPlans}, nil
}

func (s *SubscriptionService) GetMySubscription(ctx context.Context, req *pb.GetMySubscriptionRequest) (*pb.GetMySubscriptionReply, error) {
	sub, err := s.uc.GetMySubscription(ctx, req.Uid)
	if err != nil {
		return nil, err
	}

	if sub == nil {
		return &pb.GetMySubscriptionReply{IsActive: false}, nil
	}

	return &pb.GetMySubscriptionReply{
		IsActive:  sub.Status == "active",
		PlanId:    sub.PlanID,
		StartTime: sub.StartTime.Unix(),
		EndTime:   sub.EndTime.Unix(),
		Status:    sub.Status,
	}, nil
}

func (s *SubscriptionService) CreateSubscriptionOrder(ctx context.Context, req *pb.CreateSubscriptionOrderRequest) (*pb.CreateSubscriptionOrderReply, error) {
	order, paymentID, payUrl, payCode, payParams, err := s.uc.CreateSubscriptionOrder(ctx, req.Uid, req.PlanId, req.PaymentMethod)
	if err != nil {
		return nil, err
	}

	return &pb.CreateSubscriptionOrderReply{
		OrderId:   order.ID,
		PaymentId: paymentID,
		PayUrl:    payUrl,
		PayCode:   payCode,
		PayParams: payParams,
	}, nil
}

func (s *SubscriptionService) HandlePaymentSuccess(ctx context.Context, req *pb.HandlePaymentSuccessRequest) (*pb.HandlePaymentSuccessReply, error) {
	err := s.uc.HandlePaymentSuccess(ctx, req.OrderId, req.Amount)
	if err != nil {
		return &pb.HandlePaymentSuccessReply{Success: false}, err
	}
	return &pb.HandlePaymentSuccessReply{Success: true}, nil
}
