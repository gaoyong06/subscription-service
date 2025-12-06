-- Migration: Add app_id field to subscription_order table
-- Date: 2024-12-06
-- Description: 添加 app_id 字段用于关联应用，便于统计应用收入

-- 添加 app_id 字段
ALTER TABLE `subscription_order` ADD COLUMN `app_id` varchar(50) DEFAULT '' COMMENT '应用ID' AFTER `plan_id`;

-- 添加索引
ALTER TABLE `subscription_order` ADD INDEX `idx_app_id` (`app_id`);

-- 可选：为现有数据设置 app_id（通过 plan 表关联查询）
-- UPDATE `subscription_order` so
-- INNER JOIN `plan` p ON so.plan_id = p.plan_id
-- SET so.app_id = p.app_id
-- WHERE so.app_id = '';

