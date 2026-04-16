-- Rollback migration for resources schema

-- Drop views
DROP VIEW IF EXISTS optimization_candidates;
DROP VIEW IF EXISTS resource_summary;

-- Drop functions
DROP FUNCTION IF EXISTS get_resource_statistics();
DROP FUNCTION IF EXISTS search_resources_by_tag(TEXT, TEXT);

-- Drop trigger
DROP TRIGGER IF EXISTS update_resources_updated_at ON resources;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_resources_optimization_candidates;
DROP INDEX IF EXISTS idx_resources_production;
DROP INDEX IF EXISTS idx_resources_active_ebs;
DROP INDEX IF EXISTS idx_resources_active_rds;
DROP INDEX IF EXISTS idx_resources_active_ec2;
DROP INDEX IF EXISTS idx_resources_updated_at;
DROP INDEX IF EXISTS idx_resources_created_at;
DROP INDEX IF EXISTS idx_resources_state_type;
DROP INDEX IF EXISTS idx_resources_account_type;
DROP INDEX IF EXISTS idx_resources_region_type;
DROP INDEX IF EXISTS idx_resources_provider_type;
DROP INDEX IF EXISTS idx_resources_instance_type;
DROP INDEX IF EXISTS idx_resources_state;
DROP INDEX IF EXISTS idx_resources_account_id;
DROP INDEX IF EXISTS idx_resources_region;
DROP INDEX IF EXISTS idx_resources_provider;
DROP INDEX IF EXISTS idx_resources_resource_type;
DROP INDEX IF EXISTS idx_resources_metadata_gin;
DROP INDEX IF EXISTS idx_resources_tags_gin;

-- Drop table
DROP TABLE IF EXISTS resources;
