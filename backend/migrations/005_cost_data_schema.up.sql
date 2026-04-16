-- Migration: 005_cost_data_schema.up.sql
-- Creates the cost_data table for storing AWS Cost Explorer data
-- with proper indexing and constraints for efficient querying

-- Create cost_data table
CREATE TABLE IF NOT EXISTS cost_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Resource identification (links to resources table)
    resource_id VARCHAR(255) NOT NULL,
    resource_uuid UUID REFERENCES resources(id) ON DELETE CASCADE,
    
    -- Cost information
    service VARCHAR(100) NOT NULL,
    cost_amount DECIMAL(15, 6) NOT NULL DEFAULT 0.0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    
    -- Time period information
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Additional metadata
    usage_type VARCHAR(255),
    region VARCHAR(50),
    account_id VARCHAR(50),
    
    -- Raw data storage for flexibility
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT cost_data_amount_positive CHECK (cost_amount >= 0),
    CONSTRAINT cost_data_date_range_valid CHECK (start_date <= end_date),
    CONSTRAINT cost_data_currency_length CHECK (LENGTH(currency) = 3)
);

-- Create unique constraint for upsert operations
-- Prevents duplicate cost entries for same resource, service, and date range
CREATE UNIQUE INDEX IF NOT EXISTS idx_cost_data_unique 
ON cost_data (resource_id, service, start_date, end_date, region);

-- Indexes for efficient querying

-- Index for resource_id lookups (foreign key relationship)
CREATE INDEX IF NOT EXISTS idx_cost_data_resource_id 
ON cost_data(resource_id);

-- Index for resource_uuid lookups (foreign key relationship)
CREATE INDEX IF NOT EXISTS idx_cost_data_resource_uuid 
ON cost_data(resource_uuid);

-- Index for service-based queries
CREATE INDEX IF NOT EXISTS idx_cost_data_service 
ON cost_data(service);

-- Index for date range queries (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_cost_data_date_range 
ON cost_data(start_date, end_date);

-- Index for timestamp-based queries
CREATE INDEX IF NOT EXISTS idx_cost_data_timestamp 
ON cost_data(timestamp DESC);

-- Composite index for resource + date queries
CREATE INDEX IF NOT EXISTS idx_cost_data_resource_date 
ON cost_data(resource_id, start_date, end_date);

-- Composite index for service + date queries
CREATE INDEX IF NOT EXISTS idx_cost_data_service_date 
ON cost_data(service, start_date, end_date);

-- Index for account-based queries
CREATE INDEX IF NOT EXISTS idx_cost_data_account 
ON cost_data(account_id);

-- Index for region-based queries
CREATE INDEX IF NOT EXISTS idx_cost_data_region 
ON cost_data(region);

-- Partial index for high-cost entries (optimization candidates)
CREATE INDEX IF NOT EXISTS idx_cost_data_high_cost 
ON cost_data(resource_id, cost_amount DESC) 
WHERE cost_amount > 100.0;

-- Index on metadata JSONB for flexible queries
CREATE INDEX IF NOT EXISTS idx_cost_data_metadata_gin 
ON cost_data USING GIN(metadata);

-- Create function for automatic updated_at timestamp
CREATE OR REPLACE FUNCTION update_cost_data_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for updated_at
CREATE TRIGGER cost_data_updated_at_trigger
    BEFORE UPDATE ON cost_data
    FOR EACH ROW
    EXECUTE FUNCTION update_cost_data_updated_at();

-- Create view for cost summary by service
CREATE OR REPLACE VIEW cost_summary_by_service AS
SELECT 
    service,
    start_date,
    end_date,
    account_id,
    region,
    COUNT(*) as record_count,
    SUM(cost_amount) as total_cost,
    AVG(cost_amount) as avg_cost,
    MIN(cost_amount) as min_cost,
    MAX(cost_amount) as max_cost,
    currency
FROM cost_data
GROUP BY service, start_date, end_date, account_id, region, currency
ORDER BY total_cost DESC;

-- Create view for cost summary by resource
CREATE OR REPLACE VIEW cost_summary_by_resource AS
SELECT 
    resource_id,
    resource_uuid,
    service,
    account_id,
    region,
    COUNT(*) as record_count,
    SUM(cost_amount) as total_cost,
    AVG(cost_amount) as avg_daily_cost,
    MIN(start_date) as first_seen,
    MAX(end_date) as last_seen,
    currency
FROM cost_data
GROUP BY resource_id, resource_uuid, service, account_id, region, currency
ORDER BY total_cost DESC;

-- Create materialized view for expensive resource analysis
CREATE MATERIALIZED VIEW expensive_resources AS
SELECT 
    resource_id,
    service,
    account_id,
    region,
    SUM(cost_amount) as total_cost,
    COUNT(*) as billing_days,
    AVG(cost_amount) as avg_daily_cost,
    currency,
    MAX(created_at) as last_updated
FROM cost_data
GROUP BY resource_id, service, account_id, region, currency
HAVING SUM(cost_amount) > 100.0
ORDER BY total_cost DESC;

-- Create index on materialized view
CREATE INDEX idx_expensive_resources_cost 
ON expensive_resources(total_cost DESC);

-- Create function to refresh materialized view
CREATE OR REPLACE FUNCTION refresh_expensive_resources()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY expensive_resources;
END;
$$ LANGUAGE plpgsql;

-- Add table comments for documentation
COMMENT ON TABLE cost_data IS 'Stores AWS Cost Explorer cost data linked to resources';
COMMENT ON COLUMN cost_data.resource_id IS 'AWS resource identifier (e.g., i-1234567890abcdef0)';
COMMENT ON COLUMN cost_data.resource_uuid IS 'Foreign key to resources table for relational queries';
COMMENT ON COLUMN cost_data.service IS 'AWS service name (e.g., Amazon EC2, Amazon RDS)';
COMMENT ON COLUMN cost_data.cost_amount IS 'Cost amount in the specified currency';
COMMENT ON COLUMN cost_data.start_date IS 'Billing period start date';
COMMENT ON COLUMN cost_data.end_date IS 'Billing period end date';
COMMENT ON COLUMN cost_data.metadata IS 'Additional JSON metadata for flexibility';
