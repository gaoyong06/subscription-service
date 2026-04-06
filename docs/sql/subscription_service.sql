/*
 Navicat Premium Dump SQL

 Source Server         : localhost
 Source Server Type    : MySQL
 Source Server Version : 90500 (9.5.0)
 Source Host           : localhost:3306
 Source Schema         : subscription_service

 Target Server Type    : MySQL
 Target Server Version : 90500 (9.5.0)
 File Encoding         : 65001

 Date: 22/01/2026 11:45:51
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for plan
-- ----------------------------
DROP TABLE IF EXISTS `plan`;
CREATE TABLE `plan` (
  `plan_id` varchar(50) NOT NULL COMMENT '套餐ID',
  `app_id` varchar(36) NOT NULL COMMENT '应用ID（关联api-key-service的app表）',
  `user_id` varchar(36) NOT NULL COMMENT '开发者ID（用户ID，关联api-key-service的app.user_id）',
  `name` varchar(100) NOT NULL COMMENT '套餐名称',
  `description` varchar(255) DEFAULT '' COMMENT '描述',
  `price` decimal(10,2) NOT NULL COMMENT '默认价格（用于兜底，如果plan_pricing表中没有对应地域的价格）',
  `currency` varchar(10) NOT NULL DEFAULT 'USD' COMMENT '默认币种（用于兜底）',
  `period_type` varchar(16) NOT NULL DEFAULT 'MONTH' COMMENT '计费周期: DAY/MONTH/YEAR/FOREVER（UTC自然历）',
  `interval_count` int NOT NULL DEFAULT 1 COMMENT '周期倍数；FOREVER 为 0',
  `features` json DEFAULT NULL COMMENT '权益 i18n key 数组（JSON，空为[]）',
  `type` varchar(20) NOT NULL COMMENT '类型',
  `deleted_at` datetime(3) DEFAULT NULL COMMENT '软删除时间，非空表示已删除',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`plan_id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_app_user_id` (`app_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订阅套餐表（每个app可以设置不同的套餐）';

-- ----------------------------
-- Table structure for plan_pricing
-- ----------------------------
DROP TABLE IF EXISTS `plan_pricing`;
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
  UNIQUE KEY `uk_plan_country` (`plan_id`,`country_code`),
  KEY `idx_plan_id` (`plan_id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_app_plan_country` (`app_id`,`plan_id`,`country_code`),
  KEY `idx_country_code` (`country_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='套餐区域定价表（所有价格都在数据库中配置，支持按地域定价）';

-- ----------------------------
-- Table structure for subscription_history
-- ----------------------------
DROP TABLE IF EXISTS `subscription_history`;
CREATE TABLE `subscription_history` (
  `subscription_history_id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '历史记录ID',
  `user_id` varchar(36) NOT NULL COMMENT '用户ID',
  `plan_id` varchar(50) NOT NULL COMMENT '套餐ID',
  `plan_name` varchar(100) NOT NULL COMMENT '套餐名称',
  `app_id` varchar(50) NOT NULL DEFAULT '' COMMENT '应用ID（冗余字段，通过plan_id关联，便于按app统计和查询）',
  `start_time` datetime NOT NULL COMMENT '开始时间',
  `end_time` datetime NOT NULL COMMENT '结束时间',
  `status` varchar(20) NOT NULL COMMENT '状态',
  `action` enum('created','renewed','upgraded','paused','resumed','cancelled','expired','enabled_auto_renew','disabled_auto_renew') NOT NULL COMMENT '操作类型: created-创建, renewed-续费, upgraded-升级, paused-暂停, resumed-恢复, cancelled-取消, expired-过期, enabled_auto_renew-启用自动续费, disabled_auto_renew-禁用自动续费',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`subscription_history_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_app_user_id` (`app_id`,`user_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订阅历史记录表';

-- ----------------------------
-- Table structure for subscription_order
-- ----------------------------
DROP TABLE IF EXISTS `subscription_order`;
CREATE TABLE `subscription_order` (
  `order_id` varchar(64) NOT NULL COMMENT '订单号(业务订单号，与payment-service的order_id相同)',
  `payment_id` varchar(19) DEFAULT '' COMMENT '支付流水号(payment-service返回的payment_id，用于追溯支付记录)',
  `user_id` varchar(36) NOT NULL COMMENT '用户ID',
  `plan_id` varchar(50) NOT NULL COMMENT '套餐ID',
  `app_id` varchar(50) DEFAULT '' COMMENT '应用ID',
  `amount` decimal(10,2) NOT NULL COMMENT '金额',
  `payment_status` enum('pending','success','failed','closed','refunded','partially_refunded') NOT NULL DEFAULT 'pending' COMMENT '支付状态(与payment-service保持一致): pending-待支付(订单已创建，等待支付), success-支付成功, failed-支付失败, closed-订单关闭, refunded-已全额退款, partially_refunded-部分退款',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`order_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_payment_id` (`payment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订阅订单表';

-- ----------------------------
-- Table structure for user_subscription
-- ----------------------------
DROP TABLE IF EXISTS `user_subscription`;
CREATE TABLE `user_subscription` (
  `subscription_id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '订阅ID',
  `user_id` varchar(36) NOT NULL COMMENT '用户ID',
  `plan_id` varchar(50) NOT NULL COMMENT '当前套餐ID',
  `app_id` varchar(50) NOT NULL DEFAULT '' COMMENT '应用ID（冗余字段，通过plan_id关联，便于按app统计和查询）',
  `billing_anchor_day` tinyint NOT NULL DEFAULT 0 COMMENT '自然月/年锚点日1-31；DAY/FOREVER为0',
  `start_time` datetime NOT NULL COMMENT '开始时间',
  `end_time` datetime NOT NULL COMMENT '结束时间',
  `status` enum('active','expired','paused','cancelled') NOT NULL DEFAULT 'active' COMMENT '订阅状态: active-活跃(订阅有效中), expired-过期(订阅已过期), paused-暂停(用户主动暂停), cancelled-已取消(用户主动取消)',
  `order_id` varchar(64) NOT NULL DEFAULT '' COMMENT '订单ID（关联subscription_order表）',
  `is_auto_renew` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否自动续费',
  `created_at` datetime DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`subscription_id`),
  UNIQUE KEY `idx_user_id` (`user_id`),
  KEY `idx_app_id` (`app_id`),
  KEY `idx_app_user_id` (`app_id`,`user_id`),
  KEY `idx_end_time` (`end_time`),
  KEY `idx_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户订阅表';

SET FOREIGN_KEY_CHECKS = 1;
