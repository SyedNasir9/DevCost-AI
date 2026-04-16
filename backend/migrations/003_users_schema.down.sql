-- Rollback migration for DevCost AI Users Schema
-- Migration: 003_users_schema (down)
-- Description: Drop user management and cloud account tables

-- Drop indexes
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_last_login;

DROP INDEX IF EXISTS idx_cloud_accounts_user_id;
DROP INDEX IF EXISTS idx_cloud_accounts_provider;
DROP INDEX IF EXISTS idx_cloud_accounts_is_active;
DROP INDEX IF EXISTS idx_cloud_accounts_last_sync;
DROP INDEX IF EXISTS idx_cloud_accounts_sync_status;

DROP INDEX IF EXISTS idx_cost_alerts_user_id;
DROP INDEX IF EXISTS idx_cost_alerts_cloud_account_id;
DROP INDEX IF EXISTS idx_cost_alerts_type;
DROP INDEX IF EXISTS idx_cost_alerts_is_active;
DROP INDEX IF EXISTS idx_cost_alerts_last_triggered;

DROP INDEX IF EXISTS idx_alert_notifications_alert_id;
DROP INDEX IF EXISTS idx_alert_notifications_channel;
DROP INDEX IF EXISTS idx_alert_notifications_status;
DROP INDEX IF EXISTS idx_alert_notifications_sent_at;

-- Drop tables in correct order
DROP TABLE IF EXISTS alert_notifications;
DROP TABLE IF EXISTS cost_alerts;
DROP TABLE IF EXISTS cloud_accounts;
DROP TABLE IF EXISTS users;
