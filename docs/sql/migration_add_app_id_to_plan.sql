-- Add app_id to plan table
ALTER TABLE `plan` ADD COLUMN `app_id` VARCHAR(50) NOT NULL DEFAULT 'default' COMMENT '应用ID' AFTER `plan_id`;
ALTER TABLE `plan` ADD INDEX `idx_app_id` (`app_id`);

-- Update existing plans to have a default app_id if needed (optional)
-- UPDATE `plan` SET `app_id` = 'default' WHERE `app_id` IS NULL;
