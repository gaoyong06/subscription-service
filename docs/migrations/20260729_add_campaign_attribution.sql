-- 保存订单创建时的投放归因快照，避免支付完成后依赖当前页面反推来源。
ALTER TABLE `subscription_order`
  ADD COLUMN `campaign_id` varchar(64) DEFAULT NULL COMMENT '投放活动 ID' AFTER `app_id`,
  ADD COLUMN `click_id` varchar(64) DEFAULT NULL COMMENT '短链点击 ID' AFTER `campaign_id`,
  ADD KEY `idx_campaign_id` (`campaign_id`),
  ADD KEY `idx_click_id` (`click_id`);
