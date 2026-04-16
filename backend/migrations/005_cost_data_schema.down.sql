-- Migration: 005_cost_data_schema.down.sql
-- Rolls back the cost_data schema changes

-- Drop materialized view
DROP MATERIALIZED VIEW IF EXISTS expensive_resources CASCADE;

-- Drop views
DROP VIEW IF EXISTS cost_summary_by_resource CASCADE;
DROP VIEW IF EXISTS cost_summary_by_service CASCADE;

-- Drop trigger
DROP TRIGGER IF EXISTS cost_data_updated_at_trigger ON cost_data;

-- Drop function
DROP FUNCTION IF EXISTS update_cost_data_updated_at();
DROP FUNCTION IF EXISTS refresh_expensive_resources();

-- Drop table (this will automatically drop all indexes)
DROP TABLE IF EXISTS cost_data CASCADE;
