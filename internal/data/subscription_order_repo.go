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
		ID:            order.ID,
		UserID:        order.UserID,
		PlanID:        order.PlanID,
		Amount:        order.Amount,
		PaymentStatus: order.PaymentStatus,
		CreatedAt:     order.CreatedAt,
	}
	if err := r.data.db.WithContext(ctx).Create(m).Error; err != nil {
		r.log.Errorf("Failed to create order %s: %v", order.ID, err)
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
		ID:            m.ID,
		UserID:        m.UserID,
		PlanID:        m.PlanID,
		Amount:        m.Amount,
		PaymentStatus: m.PaymentStatus,
		CreatedAt:     m.CreatedAt,
	}, nil
}

// UpdateOrder 更新订单
func (r *orderRepo) UpdateOrder(ctx context.Context, order *biz.SubscriptionOrder) error {
	m := &model.SubscriptionOrder{
		ID:            order.ID,
		UserID:        order.UserID,
		PlanID:        order.PlanID,
		Amount:        order.Amount,
		PaymentStatus: order.PaymentStatus,
		CreatedAt:     order.CreatedAt,
	}
	if err := r.data.db.WithContext(ctx).Save(m).Error; err != nil {
		r.log.Errorf("Failed to update order %s: %v", order.ID, err)
		return err
	}
	return nil
}
