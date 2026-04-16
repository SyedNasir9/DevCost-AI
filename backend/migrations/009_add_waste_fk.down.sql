-- Migration: 009_add_waste_fk.down.sql
-- Removes the waste_id foreign key from recommendations

ALTER TABLE recommendations 
DROP CONSTRAINT IF EXISTS fk_recommendations_waste_id;
