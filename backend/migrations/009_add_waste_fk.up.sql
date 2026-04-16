-- Migration: 009_add_waste_fk.up.sql
-- Adds foreign key from recommendations to waste_results
-- Run after 008_waste_results_schema.up.sql

-- Add foreign key constraint
ALTER TABLE recommendations 
ADD CONSTRAINT fk_recommendations_waste_id 
FOREIGN KEY (waste_id) 
REFERENCES waste_results(id) 
ON DELETE SET NULL;

-- Index on waste_id (if not exists)
CREATE INDEX IF NOT EXISTS idx_recommendations_waste_id ON recommendations(waste_id);
