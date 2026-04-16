-- Migration: 006_recommendations_schema.down.sql
-- Rolls back the recommendations schema

-- Drop views
DROP VIEW IF EXISTS high_impact_recommendations CASCADE;
DROP VIEW IF EXISTS recommendations_summary_by_type CASCADE;
DROP VIEW IF EXISTS active_recommendations_view CASCADE;

-- Drop materialized view
DROP MATERIALIZED VIEW IF EXISTS recommendation_analytics CASCADE;

-- Drop function
DROP FUNCTION IF EXISTS refresh_recommendation_analytics() CASCADE;
DROP FUNCTION IF EXISTS expire_old_recommendations(INT) CASCADE;

-- Drop trigger
DROP TRIGGER IF EXISTS recommendations_updated_at_trigger ON recommendations;

-- Drop function
DROP FUNCTION IF EXISTS update_recommendations_updated_at() CASCADE;

-- Drop table (this will automatically drop all indexes)
DROP TABLE IF EXISTS recommendations CASCADE;
