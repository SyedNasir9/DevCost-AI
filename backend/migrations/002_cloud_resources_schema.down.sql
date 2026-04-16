-- Rollback migration for DevCost AI Cloud Resources Schema
-- Migration: 002_cloud_resources_schema (down)
-- Description: Drop tables for cloud resources, usage tracking, cost data, recommendations, and actions

-- Drop indexes first
DROP INDEX IF EXISTS idx_resources_provider_region_type;
DROP INDEX IF EXISTS idx_resources_provider_region_type;
DROP INDEX IF EXISTS idx_resources_resource_id;
DROP INDEX IF EXISTS idx_resources_provider;
DROP INDEX IF EXISTS idx_resources_region;
DROP INDEX IF EXISTS idx_resources_account_id;
DROP INDEX IF EXISTS idx_resources_resource_type;
DROP INDEX IF EXISTS idx_resources_state;
DROP INDEX IF EXISTS idx_resources_created_at;
DROP INDEX IF EXISTS idx_resources_tags_gin;

DROP INDEX IF EXISTS idx_resource_usage_resource_id;
DROP INDEX IF EXISTS idx_resource_usage_timestamp;
DROP INDEX IF EXISTS idx_resource_usage_resource_timestamp;

DROP INDEX IF EXISTS idx_cost_data_resource_id;
DROP INDEX IF EXISTS idx_cost_data_billing_period;
DROP INDEX IF EXISTS idx_cost_data_service_name;
DROP INDEX IF EXISTS idx_cost_data_cost_amount;
DROP INDEX IF EXISTS idx_cost_data_created_at;
DROP INDEX IF EXISTS idx_cost_data_tags_gin;

DROP INDEX IF EXISTS idx_recommendations_resource_id;
DROP INDEX IF EXISTS idx_recommendations_type;
DROP INDEX IF EXISTS idx_recommendations_status;
DROP INDEX IF EXISTS idx_recommendations_priority;
DROP INDEX IF EXISTS idx_recommendations_created_at;
DROP INDEX IF EXISTS idx_recommendations_expires_at;

DROP INDEX IF EXISTS idx_actions_resource_id;
DROP INDEX IF EXISTS idx_actions_type;
DROP INDEX IF EXISTS idx_actions_status;
DROP INDEX IF EXISTS idx_actions_created_at;
DROP INDEX IF EXISTS idx_actions_scheduled_at;

DROP INDEX IF EXISTS idx_cost_data_resource_period;
DROP INDEX IF EXISTS idx_recommendations_resource_status;

-- Drop tables in correct order (respecting foreign key dependencies)
DROP TABLE IF EXISTS actions;
DROP TABLE IF EXISTS recommendations;
DROP TABLE IF EXISTS cost_data;
DROP TABLE IF EXISTS resource_usage;
DROP TABLE IF EXISTS resources;

-- Note: Extensions are not dropped as they might be used by other tables
