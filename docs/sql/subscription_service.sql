CREATE TABLE `plan` (
  `plan_id` varchar(50) NOT NULL COMMENT '套餐ID',
  `name` varchar(100) NOT NULL COMMENT '套餐名称',
  `description` varchar(255) DEFAULT '' COMMENT '描述',
  `price` decimal(10,2) NOT NULL COMMENT '价格',
  `currency` varchar(10) NOT NULL DEFAULT 'CNY' COMMENT '币种',
  `duration_days` int NOT NULL COMMENT '持续天数',
  `type` varchar(20) NOT NULL COMMENT '类型',
  PRIMARY KEY (`plan_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订阅套餐表';

CREATE TABLE `user_subscription` (
  `user_subscription_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `plan_id` varchar(50) NOT NULL COMMENT '当前套餐ID',
  `start_time` datetime NOT NULL COMMENT '开始时间',
  `end_time` datetime NOT NULL COMMENT '结束时间',
  `status` varchar(20) NOT NULL COMMENT '状态',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_subscription_id`),
  UNIQUE KEY `idx_user_id` (`user_id`),
  KEY `idx_end_time` (`end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户订阅表';

CREATE TABLE `subscription_order` (
  `order_id` varchar(64) NOT NULL COMMENT '订单号',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `plan_id` varchar(50) NOT NULL COMMENT '套餐ID',
  `amount` decimal(10,2) NOT NULL COMMENT '金额',
  `payment_status` varchar(20) NOT NULL COMMENT '支付状态',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`order_id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订阅订单表';

-- 初始化数据
INSERT INTO `plan` (`plan_id`, `name`, `description`, `price`, `currency`, `duration_days`, `type`) VALUES
('plan_monthly', 'Pro Monthly', 'Pro features for 1 month', 9.99, 'CNY', 30, 'pro'),
('plan_yearly', 'Pro Yearly', 'Pro features for 1 year', 99.99, 'CNY', 365, 'pro');
