package data

import (
	"context"
	"xinyuan_tech/subscription-service/internal/biz"
	"xinyuan_tech/subscription-service/internal/data/model"

	"github.com/go-kratos/kratos/v2/log"
)

// orderRepo 订单仓库实现
type orderRepo struct {
	data *Data
	log  *log.Helper
}

// NewSubscriptionOrderRepo 创建订阅订单仓库
func NewSubscriptionOrderRepo(data *Data, logger log.Logger) biz.SubscriptionOrderRepo {
	return &orderRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// CreateOrder 创建订单
func (r *orderRepo) CreateOrder(ctx context.Context, order *biz.SubscriptionOrder) error {
	m := &model.SubscriptionOrder{
		OrderID:       order.OrderID,
		PaymentID:     order.PaymentID,
		UserID:        order.UserID,
		PlanID:        order.PlanID,
		AppID:         order.AppID,
		Amount:        order.Amount,
		PaymentStatus: order.PaymentStatus,
		CreatedAt:     order.CreatedAt,
	}
	if err := r.data.db.WithContext(ctx).Create(m).Error; err != nil {
		r.log.Errorf("Failed to create order %s: %v", order.OrderID, err)
		return err
	}
	return nil
}

// GetOrder 获取订单
func (r *orderRepo) GetOrder(ctx context.Context, orderID string) (*biz.SubscriptionOrder, error) {
	var m model.SubscriptionOrder
	if err := r.data.db.WithContext(ctx).First(&m, "order_id = ?", orderID).Error; err != nil {
		r.log.Errorf("Failed to get order %s: %v", orderID, err)
		return nil, err
	}
	return &biz.SubscriptionOrder{
		OrderID:       m.OrderID,
		PaymentID:     m.PaymentID,
		UserID:        m.UserID,
		PlanID:        m.PlanID,
		AppID:         m.AppID,
		Amount:        m.Amount,
		PaymentStatus: m.PaymentStatus,
		CreatedAt:     m.CreatedAt,
	}, nil
}

// UpdateOrder 更新订单
func (r *orderRepo) UpdateOrder(ctx context.Context, order *biz.SubscriptionOrder) error {
	m := &model.SubscriptionOrder{
		OrderID:       order.OrderID,
		PaymentID:     order.PaymentID,
		UserID:        order.UserID,
		PlanID:        order.PlanID,
		AppID:         order.AppID,
		Amount:        order.Amount,
		PaymentStatus: order.PaymentStatus,
		CreatedAt:     order.CreatedAt,
	}
	if err := r.data.db.WithContext(ctx).Save(m).Error; err != nil {
		r.log.Errorf("Failed to update order %s: %v", order.OrderID, err)
		return err
	}
	return nil
}

// ListOrders 查询订阅订单列表
func (r *orderRepo) ListOrders(ctx context.Context, appID, userID, planID, status string, page, pageSize int) ([]*biz.SubscriptionOrder, int, error) {
	var models []model.SubscriptionOrder
	var total int64

	// 构建查询条件
	query := r.data.db.WithContext(ctx).Model(&model.SubscriptionOrder{})

	if appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if planID != "" {
		query = query.Where("plan_id = ?", planID)
	}
	if status != "" {
		query = query.Where("payment_status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		r.log.Errorf("Failed to count subscription orders: %v", err)
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&models).Error; err != nil {
		r.log.Errorf("Failed to list subscription orders: %v", err)
		return nil, 0, err
	}

	// 转换为业务对象
	orders := make([]*biz.SubscriptionOrder, len(models))
	for i, m := range models {
		orders[i] = &biz.SubscriptionOrder{
			OrderID:       m.OrderID,
			PaymentID:     m.PaymentID,
			UserID:        m.UserID,
			PlanID:        m.PlanID,
			AppID:         m.AppID,
			Amount:        m.Amount,
			PaymentStatus: m.PaymentStatus,
			CreatedAt:     m.CreatedAt,
		}
	}

	return orders, int(total), nil
}
