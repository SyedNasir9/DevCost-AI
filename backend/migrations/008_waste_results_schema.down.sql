-- Migration: 008_waste_results_schema.down.sql
-- Removes the waste_results table

DROP VIEW IF EXISTS active_waste_view;
DROP TRIGGER IF EXISTS waste_results_updated_at_trigger ON waste_results;
DROP FUNCTION IF EXISTS update_waste_results_updated_at();
DROP TABLE IF EXISTS waste_results;
