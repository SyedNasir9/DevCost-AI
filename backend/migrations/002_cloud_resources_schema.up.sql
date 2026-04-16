-- DevCost AI Cloud Resources Schema Migration
-- Migration: 002_cloud_resources_schema
-- Created: 2026-03-26
-- Description: Create tables for cloud resources, usage tracking, cost data, recommendations, and actions

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Resources table - stores cloud resource information
CREATE TABLE IF NOT EXISTS resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id VARCHAR(255) NOT NULL, -- Cloud provider specific resource ID (e.g., i-1234567890abcdef0)
    resource_type VARCHAR(100) NOT NULL, -- EC2, RDS, EBS, Lambda, etc.
    provider VARCHAR(50) NOT NULL, -- aws, gcp, azure
    region VARCHAR(50) NOT NULL,
    account_id VARCHAR(255) NOT NULL, -- Cloud account identifier
    name VARCHAR(255), -- Resource name
    state VARCHAR(50), -- running, stopped, terminated, etc.
    instance_type VARCHAR(100), -- t3.micro, db.t3.medium, etc.
    tags JSONB, -- Flexible storage for resource tags
    metadata JSONB, -- Additional resource metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT resources_provider_check CHECK (provider IN ('aws', 'gcp', 'azure')),
    CONSTRAINT resources_resource_type_check CHECK (
        resource_type IN (
            'ec2', 'rds', 'ebs', 'lambda', 's3', 'vpc', 'subnet', 
            'security_group', 'load_balancer', 'cloudfront', 'route53',
            'compute_engine', 'cloud_sql', 'cloud_storage', 'functions',
            'virtual_machine', 'sql_database', 'storage_account', 'app_service'
        )
    )
);

-- Resource usage metrics table
CREATE TABLE IF NOT EXISTS resource_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    cpu_usage DECIMAL(10,4), -- CPU utilization percentage
    memory_usage DECIMAL(10,4), -- Memory utilization percentage
    network_in DECIMAL(15,2), -- Network traffic in (bytes)
    network_out DECIMAL(15,2), -- Network traffic out (bytes)
    disk_usage DECIMAL(15,2), -- Disk usage (bytes)
    disk_io_read DECIMAL(15,2), -- Disk read operations
    disk_io_write DECIMAL(15,2), -- Disk write operations
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT resource_usage_cpu_check CHECK (cpu_usage >= 0 AND cpu_usage <= 100),
    CONSTRAINT resource_usage_memory_check CHECK (memory_usage >= 0 AND memory_usage <= 100)
);

-- Cost data table
CREATE TABLE IF NOT EXISTS cost_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    cost_amount DECIMAL(12,4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    billing_period_start DATE NOT NULL,
    billing_period_end DATE NOT NULL,
    service_name VARCHAR(255) NOT NULL, -- AWS service name
    usage_unit VARCHAR(100), -- Hour, GB-Hours, etc.
    usage_quantity DECIMAL(15,4),
    rate DECIMAL(12,6), -- Cost per unit
    tags JSONB, -- Cost allocation tags
    raw_data JSONB, -- Raw cost data from provider
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT cost_data_currency_check CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT cost_data_cost_check CHECK (cost_amount >= 0),
    CONSTRAINT cost_data_period_check CHECK (billing_period_end >= billing_period_start)
);

-- Recommendations table
CREATE TABLE IF NOT EXISTS recommendations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- idle, rightsize, delete, schedule, etc.
    priority VARCHAR(20) DEFAULT 'medium', -- low, medium, high, critical
    title VARCHAR(255) NOT NULL,
    description TEXT,
    estimated_savings DECIMAL(12,4), -- Estimated monthly savings
    confidence_score DECIMAL(3,2), -- 0.00 to 1.00
    current_state JSONB, -- Current resource configuration
    recommended_state JSONB, -- Recommended configuration
    status VARCHAR(20) DEFAULT 'pending', -- pending, applied, dismissed, expired
    applied_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT recommendations_type_check CHECK (
        type IN ('idle', 'rightsize', 'delete', 'schedule', 'resize', 'migrate', 'cleanup')
    ),
    CONSTRAINT recommendations_priority_check CHECK (
        priority IN ('low', 'medium', 'high', 'critical')
    ),
    CONSTRAINT recommendations_status_check CHECK (
        status IN ('pending', 'applied', 'dismissed', 'expired')
    ),
    CONSTRAINT recommendations_confidence_check CHECK (
        confidence_score >= 0.00 AND confidence_score <= 1.00
    ),
    CONSTRAINT recommendations_savings_check CHECK (estimated_savings >= 0)
);

-- Actions table - tracks automated/manual actions on resources
CREATE TABLE IF NOT EXISTS actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    action_type VARCHAR(50) NOT NULL, -- stop, start, delete, resize, migrate, etc.
    status VARCHAR(20) DEFAULT 'pending', -- pending, in_progress, completed, failed
    parameters JSONB, -- Action parameters (e.g., new instance type)
    result JSONB, -- Action execution result
    error_message TEXT,
    initiated_by VARCHAR(100), -- system, user, automation
    scheduled_at TIMESTAMP WITH TIME ZONE,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT actions_type_check CHECK (
        action_type IN (
            'stop', 'start', 'restart', 'delete', 'resize', 'migrate', 
            'backup', 'restore', 'snapshot', 'modify_tags'
        )
    ),
    CONSTRAINT actions_status_check CHECK (
        status IN ('pending', 'in_progress', 'completed', 'failed', 'cancelled')
    ),
    CONSTRAINT actions_timing_check CHECK (
        (started_at IS NULL OR completed_at IS NULL OR completed_at >= started_at)
    )
);

-- Performance indexes for resources table
CREATE INDEX IF NOT EXISTS idx_resources_resource_id ON resources(resource_id);
CREATE INDEX IF NOT EXISTS idx_resources_provider ON resources(provider);
CREATE INDEX IF NOT EXISTS idx_resources_region ON resources(region);
CREATE INDEX IF NOT EXISTS idx_resources_account_id ON resources(account_id);
CREATE INDEX IF NOT EXISTS idx_resources_resource_type ON resources(resource_type);
CREATE INDEX IF NOT EXISTS idx_resources_state ON resources(state);
CREATE INDEX IF NOT EXISTS idx_resources_created_at ON resources(created_at);

-- GIN index for JSONB tags in resources (for fast tag-based queries)
CREATE INDEX IF NOT EXISTS idx_resources_tags_gin ON resources USING GIN(tags);

-- Performance indexes for resource_usage table
CREATE INDEX IF NOT EXISTS idx_resource_usage_resource_id ON resource_usage(resource_id);
CREATE INDEX IF NOT EXISTS idx_resource_usage_timestamp ON resource_usage(timestamp);
CREATE INDEX IF NOT EXISTS idx_resource_usage_resource_timestamp ON resource_usage(resource_id, timestamp);

-- Performance indexes for cost_data table
CREATE INDEX IF NOT EXISTS idx_cost_data_resource_id ON cost_data(resource_id);
CREATE INDEX IF NOT EXISTS idx_cost_data_billing_period ON cost_data(billing_period_start, billing_period_end);
CREATE INDEX IF NOT EXISTS idx_cost_data_service_name ON cost_data(service_name);
CREATE INDEX IF NOT EXISTS idx_cost_data_cost_amount ON cost_data(cost_amount);
CREATE INDEX IF NOT EXISTS idx_cost_data_created_at ON cost_data(created_at);

-- GIN index for JSONB tags in cost_data
CREATE INDEX IF NOT EXISTS idx_cost_data_tags_gin ON cost_data USING GIN(tags);

-- Performance indexes for recommendations table
CREATE INDEX IF NOT EXISTS idx_recommendations_resource_id ON recommendations(resource_id);
CREATE INDEX IF NOT EXISTS idx_recommendations_type ON recommendations(type);
CREATE INDEX IF NOT EXISTS idx_recommendations_status ON recommendations(status);
CREATE INDEX IF NOT EXISTS idx_recommendations_priority ON recommendations(priority);
CREATE INDEX IF NOT EXISTS idx_recommendations_created_at ON recommendations(created_at);
CREATE INDEX IF NOT EXISTS idx_recommendations_expires_at ON recommendations(expires_at);

-- Performance indexes for actions table
CREATE INDEX IF NOT EXISTS idx_actions_resource_id ON actions(resource_id);
CREATE INDEX IF NOT EXISTS idx_actions_type ON actions(action_type);
CREATE INDEX IF NOT EXISTS idx_actions_status ON actions(status);
CREATE INDEX IF NOT EXISTS idx_actions_created_at ON actions(created_at);
CREATE INDEX IF NOT EXISTS idx_actions_scheduled_at ON actions(scheduled_at);

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_resources_provider_region_type ON resources(provider, region, resource_type);
CREATE INDEX IF NOT EXISTS idx_cost_data_resource_period ON cost_data(resource_id, billing_period_start, billing_period_end);
CREATE INDEX IF NOT EXISTS idx_recommendations_resource_status ON recommendations(resource_id, status);

-- Add table comments for documentation
COMMENT ON TABLE resources IS 'Stores cloud resource metadata and configuration';
COMMENT ON TABLE resource_usage IS 'Tracks resource utilization metrics over time';
COMMENT ON TABLE cost_data IS 'Stores cost information for cloud resources';
COMMENT ON TABLE recommendations IS 'Stores cost optimization recommendations';
COMMENT ON TABLE actions IS 'Tracks actions taken on resources';

-- Add column comments for clarity
COMMENT ON COLUMN resources.resource_id IS 'Cloud provider-specific resource identifier';
COMMENT ON COLUMN resources.tags IS 'Resource tags stored as JSONB for flexible querying';
COMMENT ON COLUMN resource_usage.timestamp IS 'Timestamp when usage metrics were collected';
COMMENT ON COLUMN cost_data.raw_data IS 'Original cost data from cloud provider API';
COMMENT ON COLUMN recommendations.confidence_score IS 'AI confidence in recommendation accuracy (0.00-1.00)';
COMMENT ON COLUMN actions.parameters IS 'Action-specific parameters in JSON format';
COMMENT ON COLUMN actions.result IS 'Action execution results in JSON format';
