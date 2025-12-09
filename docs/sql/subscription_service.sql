CREATE TABLE `plan` (
  `plan_id` varchar(50) NOT NULL COMMENT '套餐ID',
  `app_id` varchar(36) NOT NULL COMMENT '应用ID（关联api-key-service的app表）',
  `uid` varchar(36) NOT NULL COMMENT '开发者ID（用户ID，关联api-key-service的app.uid）',
  `name` varchar(100) NOT NULL COMMENT '套餐名称',
  `description` varchar(255) DEFAULT '' COMMENT '描述',
  `price` decimal(10,2) NOT NULL COMMENT '默认价格（用于兜底，如果plan_pricing表中没有对应地域的价格）',
  `currency` varchar(10) NOT NULL DEFAULT 'USD' COMMENT '默认币种（用于兜底）',
  `duration_days` int NOT NULL COMMENT '持续天数',
  `type` varchar(20) NOT NULL COMMENT '类型',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`plan_id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_uid` (`uid`),
  KEY `idx_app_uid` (`app_id`, `uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订阅套餐表（每个app可以设置不同的套餐）';

-- 套餐区域定价表（支持按地域定价，所有价格都在数据库中配置）
CREATE TABLE `plan_pricing` (
  `plan_pricing_id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `plan_id` varchar(50) NOT NULL COMMENT '套餐ID（关联plan表）',
  `app_id` varchar(50) NOT NULL DEFAULT '' COMMENT '应用ID（冗余字段，通过plan_id关联，便于按app查询）',
  `country_code` varchar(10) NOT NULL COMMENT '国家代码（ISO 3166-1 alpha-2，如CN, US, DE等）',
  `price` decimal(10,2) NOT NULL COMMENT '价格',
  `currency` varchar(10) NOT NULL COMMENT '币种',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`plan_pricing_id`),
  UNIQUE KEY `uk_plan_country` (`plan_id`, `country_code`),
  KEY `idx_plan_id` (`plan_id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_app_plan_country` (`app_id`, `plan_id`, `country_code`),
  KEY `idx_country_code` (`country_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='套餐区域定价表（所有价格都在数据库中配置，支持按地域定价）';

CREATE TABLE `user_subscription` (
  `subscription_id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '订阅ID',
  `uid` bigint unsigned NOT NULL COMMENT '用户ID',
  `plan_id` varchar(50) NOT NULL COMMENT '当前套餐ID',
  `app_id` varchar(50) NOT NULL DEFAULT '' COMMENT '应用ID（冗余字段，通过plan_id关联，便于按app统计和查询）',
  `start_time` datetime NOT NULL COMMENT '开始时间',
  `end_time` datetime NOT NULL COMMENT '结束时间',
  `status` enum('active', 'expired', 'paused', 'cancelled') NOT NULL DEFAULT 'active' COMMENT '订阅状态: active-活跃(订阅有效中), expired-过期(订阅已过期), paused-暂停(用户主动暂停), cancelled-已取消(用户主动取消)',
  `order_id` varchar(64) NOT NULL DEFAULT '' COMMENT '订单ID（关联subscription_order表）',
  `is_auto_renew` tinyint(1) NOT NULL DEFAULT 0 COMMENT '是否自动续费',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`subscription_id`),
  UNIQUE KEY `idx_uid` (`uid`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_app_uid` (`app_id`, `uid`),
  KEY `idx_end_time` (`end_time`),
  KEY `idx_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户订阅表';

CREATE TABLE `subscription_order` (
  `order_id` varchar(64) NOT NULL COMMENT '订单号(业务订单号，与payment-service的order_id相同)',
  `payment_id` varchar(19) DEFAULT '' COMMENT '支付流水号(payment-service返回的payment_id，用于追溯支付记录)',
  `uid` bigint unsigned NOT NULL COMMENT '用户ID',
  `plan_id` varchar(50) NOT NULL COMMENT '套餐ID',
  `app_id` varchar(50) DEFAULT '' COMMENT '应用ID',
  `amount` decimal(10,2) NOT NULL COMMENT '金额',
  `payment_status` enum('pending', 'success', 'failed', 'closed', 'refunded', 'partially_refunded') NOT NULL DEFAULT 'pending' COMMENT '支付状态(与payment-service保持一致): pending-待支付(订单已创建，等待支付), success-支付成功, failed-支付失败, closed-订单关闭, refunded-已全额退款, partially_refunded-部分退款',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`order_id`),
  KEY `idx_uid` (`uid`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_payment_id` (`payment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订阅订单表';

-- 订阅历史记录表
CREATE TABLE `subscription_history` (
  `subscription_history_id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '历史记录ID',
  `uid` bigint unsigned NOT NULL COMMENT '用户ID',
  `plan_id` varchar(50) NOT NULL COMMENT '套餐ID',
  `plan_name` varchar(100) NOT NULL COMMENT '套餐名称',
  `app_id` varchar(50) NOT NULL DEFAULT '' COMMENT '应用ID（冗余字段，通过plan_id关联，便于按app统计和查询）',
  `start_time` datetime NOT NULL COMMENT '开始时间',
  `end_time` datetime NOT NULL COMMENT '结束时间',
  `status` varchar(20) NOT NULL COMMENT '状态',
  `action` enum('created', 'renewed', 'upgraded', 'paused', 'resumed', 'cancelled', 'expired', 'enabled_auto_renew', 'disabled_auto_renew') NOT NULL COMMENT '操作类型: created-创建, renewed-续费, upgraded-升级, paused-暂停, resumed-恢复, cancelled-取消, expired-过期, enabled_auto_renew-启用自动续费, disabled_auto_renew-禁用自动续费',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`subscription_history_id`),
  KEY `idx_uid` (`uid`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_app_uid` (`app_id`, `uid`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订阅历史记录表';

-- 初始化数据示例（需要根据实际app_id和uid填写）
-- INSERT INTO `plan` (`plan_id`, `app_id`, `uid`, `name`, `description`, `price`, `currency`, `duration_days`, `type`) VALUES
-- ('plan_monthly', 'app_id_here', 'uid_here', 'Pro Monthly', 'Pro features for 1 month', 9.99, 'USD', 30, 'pro'),
-- ('plan_yearly', 'app_id_here', 'uid_here', 'Pro Yearly', 'Pro features for 1 year', 99.99, 'USD', 365, 'pro');

-- 区域定价示例（基于巨无霸指数PPP的定价策略）
-- 假设 plan_id='plan_monthly', app_id='default_app', uid='default_uid'
-- INSERT INTO `plan_pricing` (`plan_id`, `country_code`, `price`, `currency`) VALUES
-- ('plan_monthly', 'CN', 59.9, 'CNY'),  -- 中国大陆
-- ('plan_monthly', 'DE', 92, 'USD'),    -- 德国（超高购买力，兜底价格）
-- ('plan_monthly', 'FR', 92, 'USD'),    -- 法国
-- ('plan_monthly', 'US', 78, 'USD'),    -- 美国（高购买力）
-- ('plan_monthly', 'KR', 78, 'USD'),    -- 韩国
-- ('plan_monthly', 'JP', 59.9, 'USD'),  -- 日本（中等购买力）
-- ('plan_monthly', 'TW', 59.9, 'USD'),   -- 台湾
-- ('plan_monthly', 'BR', 46, 'USD'),     -- 巴西（新兴市场）
-- ('plan_monthly', 'IN', 32, 'USD');     -- 印度（发展中市场）
