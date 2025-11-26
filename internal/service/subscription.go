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
		AutoRenew: sub.AutoRenew,
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

func (s *SubscriptionService) CancelSubscription(ctx context.Context, req *pb.CancelSubscriptionRequest) (*pb.CancelSubscriptionReply, error) {
	err := s.uc.CancelSubscription(ctx, req.Uid, req.Reason)
	if err != nil {
		return &pb.CancelSubscriptionReply{Success: false, Message: err.Error()}, nil
	}
	return &pb.CancelSubscriptionReply{Success: true, Message: "Subscription cancelled successfully"}, nil
}

func (s *SubscriptionService) PauseSubscription(ctx context.Context, req *pb.PauseSubscriptionRequest) (*pb.PauseSubscriptionReply, error) {
	err := s.uc.PauseSubscription(ctx, req.Uid, req.Reason)
	if err != nil {
		return &pb.PauseSubscriptionReply{Success: false, Message: err.Error()}, nil
	}
	return &pb.PauseSubscriptionReply{Success: true, Message: "Subscription paused successfully"}, nil
}

func (s *SubscriptionService) ResumeSubscription(ctx context.Context, req *pb.ResumeSubscriptionRequest) (*pb.ResumeSubscriptionReply, error) {
	err := s.uc.ResumeSubscription(ctx, req.Uid)
	if err != nil {
		return &pb.ResumeSubscriptionReply{Success: false, Message: err.Error()}, nil
	}
	return &pb.ResumeSubscriptionReply{Success: true, Message: "Subscription resumed successfully"}, nil
}

func (s *SubscriptionService) GetSubscriptionHistory(ctx context.Context, req *pb.GetSubscriptionHistoryRequest) (*pb.GetSubscriptionHistoryReply, error) {
	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	items, total, err := s.uc.GetSubscriptionHistory(ctx, req.Uid, page, pageSize)
	if err != nil {
		return nil, err
	}

	pbItems := make([]*pb.SubscriptionHistoryItem, len(items))
	for i, item := range items {
		pbItems[i] = &pb.SubscriptionHistoryItem{
			Id:        item.ID,
			PlanId:    item.PlanID,
			PlanName:  item.PlanName,
			StartTime: item.StartTime.Unix(),
			EndTime:   item.EndTime.Unix(),
			Status:    item.Status,
			Action:    item.Action,
			CreatedAt: item.CreatedAt.Unix(),
		}
	}

	return &pb.GetSubscriptionHistoryReply{
		Items:    pbItems,
		Total:    int32(total),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}

func (s *SubscriptionService) SetAutoRenew(ctx context.Context, req *pb.SetAutoRenewRequest) (*pb.SetAutoRenewReply, error) {
	err := s.uc.SetAutoRenew(ctx, req.Uid, req.AutoRenew)
	if err != nil {
		return &pb.SetAutoRenewReply{Success: false, Message: err.Error()}, nil
	}
	message := "Auto-renew disabled successfully"
	if req.AutoRenew {
		message = "Auto-renew enabled successfully"
	}
	return &pb.SetAutoRenewReply{Success: true, Message: message}, nil
}

func (s *SubscriptionService) GetExpiringSubscriptions(ctx context.Context, req *pb.GetExpiringSubscriptionsRequest) (*pb.GetExpiringSubscriptionsReply, error) {
	daysBeforeExpiry := int(req.DaysBeforeExpiry)
	if daysBeforeExpiry == 0 {
		daysBeforeExpiry = 7
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 10
	}

	subscriptions, total, err := s.uc.GetExpiringSubscriptions(ctx, daysBeforeExpiry, page, pageSize)
	if err != nil {
		return nil, err
	}

	pbSubscriptions := make([]*pb.SubscriptionInfo, len(subscriptions))
	for i, sub := range subscriptions {
		// 获取套餐信息
		plan, _ := s.uc.GetPlan(ctx, sub.PlanID)
		planName := sub.PlanID
		amount := 0.0
		if plan != nil {
			planName = plan.Name
			amount = plan.Price
		}

		pbSubscriptions[i] = &pb.SubscriptionInfo{
			Uid:       sub.UserID,
			PlanId:    sub.PlanID,
			PlanName:  planName,
			StartTime: sub.StartTime.Unix(),
			EndTime:   sub.EndTime.Unix(),
			AutoRenew: sub.AutoRenew,
			Amount:    amount,
		}
	}

	return &pb.GetExpiringSubscriptionsReply{
		Subscriptions: pbSubscriptions,
		Total:         int32(total),
		Page:          int32(page),
		PageSize:      int32(pageSize),
	}, nil
}

func (s *SubscriptionService) UpdateExpiredSubscriptions(ctx context.Context, req *pb.UpdateExpiredSubscriptionsRequest) (*pb.UpdateExpiredSubscriptionsReply, error) {
	count, uids, err := s.uc.UpdateExpiredSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateExpiredSubscriptionsReply{
		UpdatedCount: int32(count),
		UpdatedUids:  uids,
	}, nil
}

func (s *SubscriptionService) ProcessAutoRenewals(ctx context.Context, req *pb.ProcessAutoRenewalsRequest) (*pb.ProcessAutoRenewalsReply, error) {
	daysBeforeExpiry := int(req.DaysBeforeExpiry)
	if daysBeforeExpiry == 0 {
		daysBeforeExpiry = 3
	}

	totalCount, successCount, failedCount, results, err := s.uc.ProcessAutoRenewals(ctx, daysBeforeExpiry, req.DryRun)
	if err != nil {
		return nil, err
	}

	pbResults := make([]*pb.AutoRenewResult, len(results))
	for i, result := range results {
		pbResults[i] = &pb.AutoRenewResult{
			Uid:          result.UID,
			PlanId:       result.PlanID,
			Success:      result.Success,
			OrderId:      result.OrderID,
			PaymentId:    result.PaymentID,
			ErrorMessage: result.ErrorMessage,
		}
	}

	return &pb.ProcessAutoRenewalsReply{
		TotalCount:   int32(totalCount),
		SuccessCount: int32(successCount),
		FailedCount:  int32(failedCount),
		Results:      pbResults,
	}, nil
}
