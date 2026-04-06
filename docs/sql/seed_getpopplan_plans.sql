/*
  初始化 getpopplan.com（PopPlan / table-plan）默认订阅套餐
  
  说明：
  - app_id 固定为官方应用 UUID（与 table-plan-web / dev-share 默认 APP_ID 一致）
  - 文案与 table-plan-web messages/*/modules/pricing.json 对齐；features 存 i18n key（命名空间 modules.pricing 下相对路径为 plans.*.features.*）
  - user_id：创建该应用时在 api-key-service 中的开发者用户 UUID —— 执行前请将 @seed_developer_uid 改为真实值
  
  用法示例：
    mysql -u root -p subscription_service < docs/sql/seed_getpopplan_plans.sql
  
  幂等：按 plan_id 主键 UPSERT，可重复执行。
*/

SET NAMES utf8mb4;

-- 拥有应用 546b691e-14c6-4d76-a3ab-af763a38f39e 的开发者用户 UUID（请勿使用 app_id 代替）
SET @getpopplan_app_id  = '546b691e-14c6-4d76-a3ab-af763a38f39e';
SET @seed_developer_uid = '4af51d89-8594-4aa6-8be0-196ef270f243';

INSERT INTO `plan` (
  `plan_id`, `app_id`, `user_id`, `name`, `description`,
  `price`, `currency`, `period_type`, `interval_count`, `features`, `type`,
  `deleted_at`, `created_at`, `updated_at`
) VALUES (
  'gpplan-free-v1',
  @getpopplan_app_id,
  @seed_developer_uid,
  'Basic',
  'Perfect for small events and first-time users',
  0.00,
  'USD',
  'FOREVER',
  0,
  CAST('["modules.pricing.plans.free.features.1","modules.pricing.plans.free.features.2","modules.pricing.plans.free.features.3","modules.pricing.plans.free.features.4"]' AS JSON),
  'free',
  NULL,
  NOW(),
  NOW()
), (
  'gpplan-pro-v1',
  @getpopplan_app_id,
  @seed_developer_uid,
  'Professional',
  'Ideal for medium-sized weddings and events',
  29.99,
  'USD',
  'MONTH',
  1,
  CAST('["modules.pricing.plans.pro.features.1","modules.pricing.plans.pro.features.2","modules.pricing.plans.pro.features.3","modules.pricing.plans.pro.features.4","modules.pricing.plans.pro.features.5","modules.pricing.plans.pro.features.6"]' AS JSON),
  'pro',
  NULL,
  NOW(),
  NOW()
), (
  'gpplan-enterprise-v1',
  @getpopplan_app_id,
  @seed_developer_uid,
  'Enterprise',
  'For large events and corporate clients',
  99.99,
  'USD',
  'MONTH',
  1,
  CAST('["modules.pricing.plans.enterprise.features.1","modules.pricing.plans.enterprise.features.2","modules.pricing.plans.enterprise.features.3","modules.pricing.plans.enterprise.features.4","modules.pricing.plans.enterprise.features.5","modules.pricing.plans.enterprise.features.6","modules.pricing.plans.enterprise.features.7"]' AS JSON),
  'enterprise',
  NULL,
  NOW(),
  NOW()
)
ON DUPLICATE KEY UPDATE
  `app_id`         = VALUES(`app_id`),
  `user_id`        = VALUES(`user_id`),
  `name`           = VALUES(`name`),
  `description`    = VALUES(`description`),
  `price`          = VALUES(`price`),
  `currency`       = VALUES(`currency`),
  `period_type`    = VALUES(`period_type`),
  `interval_count` = VALUES(`interval_count`),
  `features`       = VALUES(`features`),
  `type`           = VALUES(`type`),
  `deleted_at`     = NULL,
  `updated_at`     = NOW();
