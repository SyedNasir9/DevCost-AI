-- Migration: 007_actions_schema.down.sql
-- Rolls back the actions schema

-- Drop views
DROP VIEW IF EXISTS failed_actions_view CASCADE;
DROP VIEW IF EXISTS action_success_rate_view CASCADE;
DROP VIEW IF EXISTS recent_actions_view CASCADE;

-- Drop trigger
DROP TRIGGER IF EXISTS actions_updated_at_trigger ON actions;

-- Drop function
DROP FUNCTION IF EXISTS update_actions_updated_at() CASCADE;

-- Drop table (this will automatically drop all indexes)
DROP TABLE IF EXISTS actions CASCADE;
