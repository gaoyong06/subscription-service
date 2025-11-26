-- 订阅服务新功能数据库迁移脚本
-- 日期: 2025-11-26
-- 功能: 添加订阅取消、暂停/恢复、历史记录、自动续费功能

-- 1. 修改 user_subscription 表，添加 auto_renew 字段
ALTER TABLE `user_subscription` 
ADD COLUMN `auto_renew` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否自动续费' AFTER `status`;

-- 2. 修改 user_subscription 表的 status 字段注释，增加新状态
ALTER TABLE `user_subscription` 
MODIFY COLUMN `status` VARCHAR(20) NOT NULL COMMENT '订阅状态: active-活跃, expired-过期, paused-暂停, cancelled-已取消';

-- 3. 创建订阅历史记录表
CREATE TABLE IF NOT EXISTS `subscription_history` (
  `subscription_history_id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '历史记录ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `plan_id` VARCHAR(50) NOT NULL COMMENT '套餐ID',
  `plan_name` VARCHAR(100) NOT NULL COMMENT '套餐名称',
  `start_time` DATETIME NOT NULL COMMENT '开始时间',
  `end_time` DATETIME NOT NULL COMMENT '结束时间',
  `status` VARCHAR(20) NOT NULL COMMENT '状态',
  `action` VARCHAR(50) NOT NULL COMMENT '操作类型: created-创建, renewed-续费, upgraded-升级, paused-暂停, resumed-恢复, cancelled-取消',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`subscription_history_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订阅历史记录表';

-- 4. 为现有订阅创建历史记录（可选，用于数据迁移）
INSERT INTO `subscription_history` (`user_id`, `plan_id`, `plan_name`, `start_time`, `end_time`, `status`, `action`, `created_at`)
SELECT 
  us.`user_id`,
  us.`plan_id`,
  p.`name` as `plan_name`,
  us.`start_time`,
  us.`end_time`,
  us.`status`,
  'created' as `action`,
  us.`created_at`
FROM `user_subscription` us
LEFT JOIN `plan` p ON us.`plan_id` = p.`plan_id`
WHERE NOT EXISTS (
  SELECT 1 FROM `subscription_history` sh 
  WHERE sh.`user_id` = us.`user_id` 
  AND sh.`action` = 'created'
);

