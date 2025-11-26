package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

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

// UserSubscription 用户订阅记录
type UserSubscription struct {
	ID        uint64
	UserID    uint64
	PlanID    string
	StartTime time.Time
	EndTime   time.Time
	Status    string // active, expired
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Order 简易订单记录 (用于记录订阅购买请求)
type Order struct {
	ID            string
	UserID        uint64
	PlanID        string
	Amount        float64
	PaymentStatus string // pending, paid
	CreatedAt     time.Time
}

// SubscriptionRepo 数据层接口
type SubscriptionRepo interface {
	ListPlans(ctx context.Context) ([]*Plan, error)
	GetPlan(ctx context.Context, id string) (*Plan, error)
	GetSubscription(ctx context.Context, userID uint64) (*UserSubscription, error)
	SaveSubscription(ctx context.Context, sub *UserSubscription) error
	CreateOrder(ctx context.Context, order *Order) error
	GetOrder(ctx context.Context, orderID string) (*Order, error)
	UpdateOrder(ctx context.Context, order *Order) error
}

// PaymentClient 支付服务客户端接口 (防腐层)
type PaymentClient interface {
	CreatePayment(ctx context.Context, orderID string, userID uint64, amount float64, method, subject, returnURL string) (paymentID, payUrl, payCode, payParams string, err error)
}

// SubscriptionUsecase 订阅业务逻辑
type SubscriptionUsecase struct {
	repo          SubscriptionRepo
	paymentClient PaymentClient
	log           *log.Helper
}

func NewSubscriptionUsecase(repo SubscriptionRepo, paymentClient PaymentClient, logger log.Logger) *SubscriptionUsecase {
	return &SubscriptionUsecase{
		repo:          repo,
		paymentClient: paymentClient,
		log:           log.NewHelper(logger),
	}
}

func (uc *SubscriptionUsecase) ListPlans(ctx context.Context) ([]*Plan, error) {
	return uc.repo.ListPlans(ctx)
}

func (uc *SubscriptionUsecase) GetMySubscription(ctx context.Context, userID uint64) (*UserSubscription, error) {
	sub, err := uc.repo.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 检查是否过期
	if sub != nil && sub.EndTime.Before(time.Now().UTC()) {
		sub.Status = "expired"
		// 可以在这里异步更新数据库状态
	}

	return sub, nil
}

func (uc *SubscriptionUsecase) CreateSubscriptionOrder(ctx context.Context, userID uint64, planID, method string) (*Order, string, string, string, string, error) {
	// 1. 获取套餐信息
	plan, err := uc.repo.GetPlan(ctx, planID)
	if err != nil {
		return nil, "", "", "", "", err
	}
	if plan == nil {
		return nil, "", "", "", "", fmt.Errorf("plan not found")
	}

	// 2. 创建本地订单
	orderID := fmt.Sprintf("SUB%d%d", time.Now().UnixNano(), userID)
	order := &Order{
		ID:            orderID,
		UserID:        userID,
		PlanID:        planID,
		Amount:        plan.Price,
		PaymentStatus: "pending",
		CreatedAt:     time.Now().UTC(),
	}
	if err := uc.repo.CreateOrder(ctx, order); err != nil {
		return nil, "", "", "", "", err
	}

	// 3. 调用支付服务
	// TODO: ReturnURL 应该从配置中获取
	returnURL := "http://localhost:8080/subscription/success"
	paymentID, payUrl, payCode, payParams, err := uc.paymentClient.CreatePayment(ctx, orderID, userID, plan.Price, method, "Subscription: "+plan.Name, returnURL)
	if err != nil {
		return nil, "", "", "", "", err
	}

	return order, paymentID, payUrl, payCode, payParams, nil
}

func (uc *SubscriptionUsecase) HandlePaymentSuccess(ctx context.Context, orderID string, amount float64) error {
	// 1. 获取订单
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.PaymentStatus == "paid" {
		return nil // 幂等
	}

	// 2. 更新订单状态
	order.PaymentStatus = "paid"
	if err := uc.repo.UpdateOrder(ctx, order); err != nil {
		return err
	}

	// 3. 获取套餐时长
	plan, err := uc.repo.GetPlan(ctx, order.PlanID)
	if err != nil {
		return err
	}

	// 4. 更新或创建用户订阅
	sub, err := uc.repo.GetSubscription(ctx, order.UserID)
	now := time.Now().UTC()

	if sub == nil {
		// 新订阅
		sub = &UserSubscription{
			UserID:    order.UserID,
			PlanID:    order.PlanID,
			StartTime: now,
			EndTime:   now.AddDate(0, 0, plan.DurationDays),
			Status:    "active",
			CreatedAt: now,
			UpdatedAt: now,
		}
	} else {
		// 续费
		if sub.EndTime.Before(now) {
			sub.StartTime = now
			sub.EndTime = now.AddDate(0, 0, plan.DurationDays)
		} else {
			sub.EndTime = sub.EndTime.AddDate(0, 0, plan.DurationDays)
		}
		sub.PlanID = order.PlanID // 更新为最新购买的套餐
		sub.Status = "active"
		sub.UpdatedAt = now
	}

	return uc.repo.SaveSubscription(ctx, sub)
}
