package service

import (
	"context"
	pb "subscription-service/api/subscription/v1"
	"subscription-service/internal/auth"
	"subscription-service/internal/biz"
	"subscription-service/internal/constants"
	"time"

	pkgErrors "github.com/gaoyong06/go-pkg/errors"
	"github.com/gaoyong06/go-pkg/middleware/app_id"
	"github.com/gaoyong06/go-pkg/middleware/developer_id"
	pkgUtils "github.com/gaoyong06/go-pkg/utils"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SubscriptionService 订阅服务
type SubscriptionService struct {
	pb.UnimplementedSubscriptionServer
	uc *biz.SubscriptionUsecase
}

// NewSubscriptionService 创建订阅服务实例
func NewSubscriptionService(uc *biz.SubscriptionUsecase) *SubscriptionService {
	return &SubscriptionService{uc: uc}
}

// ListPlans 获取所有订阅套餐列表
// 返回系统中所有可用的订阅套餐信息
func (s *SubscriptionService) ListPlans(ctx context.Context, req *pb.ListPlansRequest) (*pb.ListPlansReply, error) {
	// 获取 app_id（只从 Context，由中间件从 Header 提取）
	appID := app_id.GetAppIDFromContext(ctx)
	if appID == "" {
		return nil, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	plans, err := s.uc.ListPlans(ctx, appID)
	if err != nil {
		return nil, err
	}

	pbPlans := make([]*pb.Plan, len(plans))
	for i, p := range plans {
		pbPlans[i] = &pb.Plan{
			PlanId:       p.PlanID,
			AppId:        p.AppID,
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

// CreatePlan 创建订阅套餐
func (s *SubscriptionService) CreatePlan(ctx context.Context, req *pb.CreatePlanRequest) (*pb.CreatePlanReply, error) {
	// 获取 app_id（只从 Context，由中间件从 Header 提取）
	appID := app_id.GetAppIDFromContext(ctx)
	if appID == "" {
		return nil, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	// 获取开发者 ID（从 Context，由中间件从 X-Developer-Id Header 提取）
	developerID := developer_id.GetDeveloperIDFromContext(ctx)
	if developerID == "" {
		return nil, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	plan := &biz.Plan{
		PlanID:       uuid.New().String(),
		AppID:        appID,
		UserID:       developerID, // 开发者 ID（用户 ID）
		Name:         req.Name,
		Description:  req.Description,
		Price:        req.Price,
		Currency:     req.Currency,
		DurationDays: int(req.DurationDays),
		Type:         req.Type,
	}
	if err := s.uc.CreatePlan(ctx, plan); err != nil {
		return nil, err
	}
	return &pb.CreatePlanReply{
		Plan: &pb.Plan{
			PlanId:       plan.PlanID,
			AppId:        plan.AppID,
			Name:         plan.Name,
			Description:  plan.Description,
			Price:        plan.Price,
			Currency:     plan.Currency,
			DurationDays: int32(plan.DurationDays),
			Type:         plan.Type,
		},
	}, nil
}

// UpdatePlan 更新订阅套餐
func (s *SubscriptionService) UpdatePlan(ctx context.Context, req *pb.UpdatePlanRequest) (*pb.UpdatePlanReply, error) {
	// 先获取现有套餐以保留 AppID 等未修改字段
	existing, err := s.uc.GetPlan(ctx, req.PlanId)
	if err != nil {
		return nil, err
	}

	plan := &biz.Plan{
		PlanID:       req.PlanId,
		AppID:        existing.AppID, // 保留原有的 AppID
		Name:         req.Name,
		Description:  req.Description,
		Price:        req.Price,
		Currency:     req.Currency,
		DurationDays: int(req.DurationDays),
		Type:         req.Type,
	}
	if err := s.uc.UpdatePlan(ctx, plan); err != nil {
		return nil, err
	}
	return &pb.UpdatePlanReply{
		Plan: &pb.Plan{
			PlanId:       plan.PlanID,
			AppId:        plan.AppID,
			Name:         plan.Name,
			Description:  plan.Description,
			Price:        plan.Price,
			Currency:     plan.Currency,
			DurationDays: int32(plan.DurationDays),
			Type:         plan.Type,
		},
	}, nil
}

// DeletePlan 删除订阅套餐
func (s *SubscriptionService) DeletePlan(ctx context.Context, req *pb.DeletePlanRequest) (*pb.DeletePlanReply, error) {
	if err := s.uc.DeletePlan(ctx, req.PlanId); err != nil {
		return nil, err
	}
	return &pb.DeletePlanReply{PlanId: req.PlanId}, nil
}

// ListPlanPricings 获取套餐的区域定价列表
func (s *SubscriptionService) ListPlanPricings(ctx context.Context, req *pb.ListPlanPricingsRequest) (*pb.ListPlanPricingsReply, error) {
	pricings, err := s.uc.ListPlanPricings(ctx, req.PlanId)
	if err != nil {
		return nil, err
	}

	pbPricings := make([]*pb.PlanPricing, len(pricings))
	for i, p := range pricings {
		pbPricings[i] = &pb.PlanPricing{
			PlanPricingId: p.PlanPricingID,
			PlanId:        p.PlanID,
			CountryCode:   p.CountryCode,
			Price:         p.Price,
			Currency:      p.Currency,
		}
	}

	return &pb.ListPlanPricingsReply{Pricings: pbPricings}, nil
}

// CreatePlanPricing 创建区域定价
func (s *SubscriptionService) CreatePlanPricing(ctx context.Context, req *pb.CreatePlanPricingRequest) (*pb.CreatePlanPricingReply, error) {
	pricing := &biz.PlanPricing{
		PlanID:      req.PlanId,
		CountryCode: req.CountryCode,
		Price:       req.Price,
		Currency:    req.Currency,
	}
	if err := s.uc.CreatePlanPricing(ctx, pricing); err != nil {
		return nil, err
	}
	return &pb.CreatePlanPricingReply{
		Pricing: &pb.PlanPricing{
			PlanPricingId: pricing.PlanPricingID,
			PlanId:        pricing.PlanID,
			CountryCode:   pricing.CountryCode,
			Price:         pricing.Price,
			Currency:      pricing.Currency,
		},
	}, nil
}

// UpdatePlanPricing 更新区域定价
func (s *SubscriptionService) UpdatePlanPricing(ctx context.Context, req *pb.UpdatePlanPricingRequest) (*pb.UpdatePlanPricingReply, error) {
	if err := s.uc.UpdatePlanPricing(ctx, req.PlanPricingId, req.Price, req.Currency); err != nil {
		return nil, err
	}
	// 获取更新后的完整信息
	pricing, err := s.uc.GetPlanPricingByID(ctx, req.PlanPricingId)
	if err != nil {
		return nil, err
	}
	return &pb.UpdatePlanPricingReply{
		Pricing: &pb.PlanPricing{
			PlanPricingId: pricing.PlanPricingID,
			PlanId:        pricing.PlanID,
			CountryCode:   pricing.CountryCode,
			Price:         pricing.Price,
			Currency:      pricing.Currency,
		},
	}, nil
}

// DeletePlanPricing 删除区域定价
func (s *SubscriptionService) DeletePlanPricing(ctx context.Context, req *pb.DeletePlanPricingRequest) (*pb.DeletePlanPricingReply, error) {
	if err := s.uc.DeletePlanPricing(ctx, req.PlanPricingId); err != nil {
		return nil, err
	}
	return &pb.DeletePlanPricingReply{PlanPricingId: req.PlanPricingId}, nil
}

// GetMySubscription 获取用户当前订阅信息
// 查询指定用户的当前订阅状态、套餐信息和有效期
func (s *SubscriptionService) GetMySubscription(ctx context.Context, req *pb.GetMySubscriptionRequest) (*pb.GetMySubscriptionReply, error) {
	// 权限验证: 只能查询自己的订阅或管理员可以查询所有
	if err := auth.CheckOwnership(ctx, req.UserId); err != nil {
		return nil, err
	}

	sub, err := s.uc.GetMySubscription(ctx, req.UserId)
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
		AutoRenew: sub.IsAutoRenew,
	}, nil
}

// CreateSubscriptionOrder 创建订阅订单
// 为用户创建订阅订单，调用支付服务生成支付信息
func (s *SubscriptionService) CreateSubscriptionOrder(ctx context.Context, req *pb.CreateSubscriptionOrderRequest) (*pb.CreateSubscriptionOrderReply, error) {
	// 权限验证
	if err := auth.CheckOwnership(ctx, req.UserId); err != nil {
		return nil, err
	}

	// 从请求中获取 region，如果为空则自动推断
	region := req.Region

	// 如果 region 为空，从 HTTP 请求中提取信息用于自动推断
	var clientIP, acceptLanguage, xLanguage string
	if region == "" {
		// 从 kratos transport 中提取 HTTP 信息
		if tr, ok := transport.FromServerContext(ctx); ok {
			header := tr.RequestHeader()
			clientIP = pkgUtils.GetClientIP(ctx)
			acceptLanguage = header.Get("Accept-Language")
			xLanguage = header.Get("X-Language")
		}
	}

	order, paymentID, payUrl, payCode, payParams, err := s.uc.CreateSubscriptionOrderWithContext(ctx, req.UserId, req.PlanId, req.PaymentMethod, region, clientIP, acceptLanguage, xLanguage)
	if err != nil {
		return nil, err
	}

	return &pb.CreateSubscriptionOrderReply{
		OrderId:   order.OrderID,
		PaymentId: paymentID,
		PayUrl:    payUrl,
		PayCode:   payCode,
		PayParams: payParams,
	}, nil
}

// HandlePaymentSuccess 处理支付成功回调
// 接收支付成功通知，更新订单状态，激活或续费用户订阅
func (s *SubscriptionService) HandlePaymentSuccess(ctx context.Context, req *pb.HandlePaymentSuccessRequest) (*emptypb.Empty, error) {
	err := s.uc.HandlePaymentSuccess(ctx, req.OrderId, req.Amount)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// CancelSubscription 取消订阅
// 用户主动取消订阅，订阅状态变更为已取消
func (s *SubscriptionService) CancelSubscription(ctx context.Context, req *pb.CancelSubscriptionRequest) (*emptypb.Empty, error) {
	// 权限验证
	if err := auth.CheckOwnership(ctx, req.UserId); err != nil {
		return nil, err
	}

	err := s.uc.CancelSubscription(ctx, req.UserId, req.Reason)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// PauseSubscription 暂停订阅
// 暂停用户订阅，订阅状态变更为已暂停
func (s *SubscriptionService) PauseSubscription(ctx context.Context, req *pb.PauseSubscriptionRequest) (*emptypb.Empty, error) {
	// 权限验证
	if err := auth.CheckOwnership(ctx, req.UserId); err != nil {
		return nil, err
	}

	err := s.uc.PauseSubscription(ctx, req.UserId, req.Reason)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ResumeSubscription 恢复订阅
// 恢复已暂停的订阅，订阅状态变更为激活
func (s *SubscriptionService) ResumeSubscription(ctx context.Context, req *pb.ResumeSubscriptionRequest) (*emptypb.Empty, error) {
	// 权限验证
	if err := auth.CheckOwnership(ctx, req.UserId); err != nil {
		return nil, err
	}

	err := s.uc.ResumeSubscription(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// GetSubscriptionHistory 获取订阅历史记录
// 查询用户的订阅历史记录，支持分页
func (s *SubscriptionService) GetSubscriptionHistory(ctx context.Context, req *pb.GetSubscriptionHistoryRequest) (*pb.GetSubscriptionHistoryReply, error) {
	// 权限验证
	if err := auth.CheckOwnership(ctx, req.UserId); err != nil {
		return nil, err
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}

	items, total, err := s.uc.GetSubscriptionHistory(ctx, req.UserId, page, pageSize)
	if err != nil {
		return nil, err
	}

	pbItems := make([]*pb.SubscriptionHistoryItem, len(items))
	for i, item := range items {
		pbItems[i] = &pb.SubscriptionHistoryItem{
			Id:        item.SubscriptionHistoryID,
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

// SetAutoRenew 设置自动续费
// 开启或关闭用户订阅的自动续费功能
func (s *SubscriptionService) SetAutoRenew(ctx context.Context, req *pb.SetAutoRenewRequest) (*emptypb.Empty, error) {
	// 权限验证
	if err := auth.CheckOwnership(ctx, req.UserId); err != nil {
		return nil, err
	}

	err := s.uc.SetAutoRenew(ctx, req.UserId, req.AutoRenew)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// GetExpiringSubscriptions 获取即将过期的订阅列表
// 查询指定天数内即将过期的订阅，用于定时任务提醒
func (s *SubscriptionService) GetExpiringSubscriptions(ctx context.Context, req *pb.GetExpiringSubscriptionsRequest) (*pb.GetExpiringSubscriptionsReply, error) {
	daysBeforeExpiry := int(req.DaysBeforeExpiry)
	if daysBeforeExpiry == 0 {
		daysBeforeExpiry = constants.DefaultExpiryDays
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = constants.DefaultPageSize
	}

	subscriptions, total, err := s.uc.GetExpiringSubscriptions(ctx, daysBeforeExpiry, page, pageSize)
	if err != nil {
		return nil, err
	}

	// 批量获取所有套餐信息，避免 N+1 查询
	allPlans, err := s.uc.ListPlans(ctx, "")
	planMap := make(map[string]*biz.Plan)
	if err == nil {
		for _, p := range allPlans {
			planMap[p.PlanID] = p
		}
	}

	pbSubscriptions := make([]*pb.SubscriptionInfo, len(subscriptions))
	for i, sub := range subscriptions {
		// 从内存 Map 中获取套餐信息
		plan := planMap[sub.PlanID]
		planName := sub.PlanID
		amount := 0.0
		if plan != nil {
			planName = plan.Name
			amount = plan.Price
		}

		pbSubscriptions[i] = &pb.SubscriptionInfo{
			UserId:    sub.UserID,
			PlanId:    sub.PlanID,
			PlanName:  planName,
			StartTime: sub.StartTime.Unix(),
			EndTime:   sub.EndTime.Unix(),
			AutoRenew: sub.IsAutoRenew,
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

// UpdateExpiredSubscriptions 批量更新过期订阅状态
// 定时任务调用，将已过期的订阅状态更新为expired
func (s *SubscriptionService) UpdateExpiredSubscriptions(ctx context.Context, req *pb.UpdateExpiredSubscriptionsRequest) (*pb.UpdateExpiredSubscriptionsReply, error) {
	count, userIds, err := s.uc.UpdateExpiredSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateExpiredSubscriptionsReply{
		UpdatedCount:   int32(count),
		UpdatedUserIds: userIds,
	}, nil
}

// ProcessAutoRenewals 处理自动续费
// 定时任务调用，为开启自动续费且即将过期的订阅自动创建续费订单
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
			UserId:       result.UserID,
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

// ListSubscriptionOrders 获取订阅订单列表（管理员视角）
func (s *SubscriptionService) ListSubscriptionOrders(ctx context.Context, req *pb.ListSubscriptionOrdersRequest) (*pb.ListSubscriptionOrdersReply, error) {
	// 获取 app_id（只从 Context，由中间件从 Header 提取）
	appID := app_id.GetAppIDFromContext(ctx)
	if appID == "" {
		return nil, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	// 参数验证和默认值
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 调用业务逻辑
	orders, total, err := s.uc.ListSubscriptionOrders(ctx, appID, req.UserId, req.PlanId, req.Status, page, pageSize)
	if err != nil {
		return nil, err
	}

	// 获取套餐信息（用于填充套餐名称和币种）
	planMap := make(map[string]*biz.Plan)
	if len(orders) > 0 {
		plans, err := s.uc.ListPlans(ctx, appID)
		if err == nil {
			for _, plan := range plans {
				planMap[plan.PlanID] = plan
			}
		}
	}

	// 转换为 protobuf 消息
	pbOrders := make([]*pb.SubscriptionOrderInfo, len(orders))
	for i, order := range orders {
		pbOrder := &pb.SubscriptionOrderInfo{
			OrderId:       order.OrderID,
			PaymentId:     order.PaymentID,
			UserId:        order.UserID,
			PlanId:        order.PlanID,
			AppId:         order.AppID,
			Amount:        order.Amount,
			PaymentStatus: order.PaymentStatus,
			CreatedAt:     order.CreatedAt.Unix(),
		}

		// 填充套餐信息
		if plan, ok := planMap[order.PlanID]; ok {
			pbOrder.PlanName = plan.Name
			pbOrder.Currency = plan.Currency
		}

		pbOrders[i] = pbOrder
	}

	return &pb.ListSubscriptionOrdersReply{
		Orders:   pbOrders,
		Total:    int32(total),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}

// GetSubscriptionOrder 获取订阅订单详情
func (s *SubscriptionService) GetSubscriptionOrder(ctx context.Context, req *pb.GetSubscriptionOrderRequest) (*pb.GetSubscriptionOrderReply, error) {
	// 获取 app_id（只从 Context，由中间件从 Header 提取）
	appID := app_id.GetAppIDFromContext(ctx)
	if appID == "" {
		return nil, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	// 调用业务逻辑
	order, err := s.uc.GetSubscriptionOrder(ctx, req.OrderId)
	if err != nil {
		return nil, err
	}

	// 验证订单属于当前应用
	if order.AppID != appID {
		return nil, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeNotFound)
	}

	// 获取套餐信息
	plan, err := s.uc.GetPlan(ctx, order.PlanID)
	if err != nil {
		return nil, err
	}

	// 转换为 protobuf 消息
	pbOrder := &pb.SubscriptionOrderInfo{
		OrderId:       order.OrderID,
		PaymentId:     order.PaymentID,
		UserId:        order.UserID,
		PlanId:        order.PlanID,
		PlanName:      plan.Name,
		AppId:         order.AppID,
		Amount:        order.Amount,
		Currency:      plan.Currency,
		PaymentStatus: order.PaymentStatus,
		CreatedAt:     order.CreatedAt.Unix(),
	}

	return &pb.GetSubscriptionOrderReply{
		Order: pbOrder,
	}, nil
}

// ListAppSubscriptions 获取应用的订阅用户列表（管理员视角）
func (s *SubscriptionService) ListAppSubscriptions(ctx context.Context, req *pb.ListAppSubscriptionsRequest) (*pb.ListAppSubscriptionsReply, error) {
	// 获取 app_id（只从 Context，由中间件从 Header 提取）
	appID := app_id.GetAppIDFromContext(ctx)
	if appID == "" {
		return nil, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	// 参数验证和默认值
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 调用业务逻辑
	subscriptions, total, err := s.uc.ListAppSubscriptions(ctx, appID, req.Status, req.UserId, page, pageSize)
	if err != nil {
		return nil, err
	}

	// 获取套餐信息（用于填充套餐名称）
	planMap := make(map[string]*biz.Plan)
	if len(subscriptions) > 0 {
		plans, err := s.uc.ListPlans(ctx, appID)
		if err == nil {
			for _, plan := range plans {
				planMap[plan.PlanID] = plan
			}
		}
	}

	// 转换为 protobuf 消息
	pbSubscriptions := make([]*pb.AppSubscriptionInfo, len(subscriptions))
	for i, sub := range subscriptions {
		pbSub := &pb.AppSubscriptionInfo{
			SubscriptionId: sub.SubscriptionID,
			UserId:         sub.UserID,
			PlanId:         sub.PlanID,
			AppId:          sub.AppID,
			StartTime:      sub.StartTime.Unix(),
			EndTime:        sub.EndTime.Unix(),
			Status:         sub.Status,
			OrderId:        sub.OrderID,
			AutoRenew:      sub.IsAutoRenew,
			CreatedAt:      sub.CreatedAt.Unix(),
			UpdatedAt:      sub.UpdatedAt.Unix(),
		}

		// 填充套餐名称
		if plan, ok := planMap[sub.PlanID]; ok {
			pbSub.PlanName = plan.Name
		}

		pbSubscriptions[i] = pbSub
	}

	return &pb.ListAppSubscriptionsReply{
		Subscriptions: pbSubscriptions,
		Total:         int32(total),
		Page:          int32(page),
		PageSize:      int32(pageSize),
	}, nil
}

// GetAppSubscriptionHistory 获取应用的订阅历史记录（管理员视角）
func (s *SubscriptionService) GetAppSubscriptionHistory(ctx context.Context, req *pb.GetAppSubscriptionHistoryRequest) (*pb.GetAppSubscriptionHistoryReply, error) {
	// 获取 app_id（只从 Context，由中间件从 Header 提取）
	appID := app_id.GetAppIDFromContext(ctx)
	if appID == "" {
		return nil, pkgErrors.NewBizErrorWithLang(ctx, pkgErrors.ErrCodeInvalidArgument)
	}

	// 参数验证和默认值
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 时间转换
	var startTime, endTime *time.Time
	if req.StartTime > 0 {
		t := time.Unix(req.StartTime, 0)
		startTime = &t
	}
	if req.EndTime > 0 {
		t := time.Unix(req.EndTime, 0)
		endTime = &t
	}

	// 调用业务逻辑
	items, total, err := s.uc.GetAppSubscriptionHistory(ctx, appID, req.UserId, req.Action, startTime, endTime, page, pageSize)
	if err != nil {
		return nil, err
	}

	// 转换为 protobuf 消息
	pbItems := make([]*pb.SubscriptionHistoryItem, len(items))
	for i, item := range items {
		pbItems[i] = &pb.SubscriptionHistoryItem{
			Id:        item.SubscriptionHistoryID,
			UserId:    item.UserID,
			PlanId:    item.PlanID,
			PlanName:  item.PlanName,
			StartTime: item.StartTime.Unix(),
			EndTime:   item.EndTime.Unix(),
			Status:    item.Status,
			Action:    item.Action,
			CreatedAt: item.CreatedAt.Unix(),
		}
	}

	return &pb.GetAppSubscriptionHistoryReply{
		Items:    pbItems,
		Total:    int32(total),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}
