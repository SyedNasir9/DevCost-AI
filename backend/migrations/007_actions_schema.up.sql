-- Migration: 007_actions_schema.up.sql
-- Creates the actions table for tracking executed actions

-- Create actions table
CREATE TABLE IF NOT EXISTS actions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Resource identification
    resource_id VARCHAR(255) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_name VARCHAR(255),
    
    -- Action details
    action_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    
    -- Execution tracking
    executed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    
    -- Error tracking
    error_message TEXT,
    error_code VARCHAR(100),
    
    -- Request/response details (stored as JSONB for flexibility)
    request_params JSONB DEFAULT '{}',
    response_data JSONB DEFAULT '{}',
    
    -- Related entities
    recommendation_id UUID REFERENCES recommendations(id) ON DELETE SET NULL,
    decision_id UUID,
    
    -- Audit trail
    executed_by VARCHAR(255),
    source VARCHAR(50) DEFAULT 'api', -- api, scheduler, manual
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT actions_type_check CHECK (action_type IN ('stop_ec2', 'delete_ebs', 'resize_rds', 'start_ec2', 'snapshot_ebs')),
    CONSTRAINT actions_status_check CHECK (status IN ('pending', 'in_progress', 'success', 'failed', 'cancelled', 'timeout')),
    CONSTRAINT actions_resource_id_not_empty CHECK (resource_id <> ''),
    CONSTRAINT actions_duration_positive CHECK (duration_ms >= 0 OR duration_ms IS NULL)
);

-- Indexes for efficient querying

-- Primary lookup indexes
CREATE INDEX IF NOT EXISTS idx_actions_resource_id ON actions(resource_id);
CREATE INDEX IF NOT EXISTS idx_actions_resource_type ON actions(resource_type);

-- Status tracking
CREATE INDEX IF NOT EXISTS idx_actions_status ON actions(status);

-- Action type filtering
CREATE INDEX IF NOT EXISTS idx_actions_action_type ON actions(action_type);

-- Time-based queries
CREATE INDEX IF NOT EXISTS idx_actions_executed_at ON actions(executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_actions_created_at ON actions(created_at DESC);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_actions_resource_status ON actions(resource_id, status);
CREATE INDEX IF NOT EXISTS idx_actions_type_status ON actions(action_type, status);

-- Error analysis
CREATE INDEX IF NOT EXISTS idx_actions_error_code ON actions(error_code) WHERE error_code IS NOT NULL;

-- Foreign key lookups
CREATE INDEX IF NOT EXISTS idx_actions_recommendation_id ON actions(recommendation_id);

-- Partial index for pending/in_progress actions
CREATE INDEX IF NOT EXISTS idx_actions_active ON actions(resource_id, action_type) 
WHERE status IN ('pending', 'in_progress');

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_actions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for updated_at
CREATE TRIGGER actions_updated_at_trigger
    BEFORE UPDATE ON actions
    FOR EACH ROW
    EXECUTE FUNCTION update_actions_updated_at();

-- View for recent actions with resource details
CREATE OR REPLACE VIEW recent_actions_view AS
SELECT 
    a.id,
    a.resource_id,
    a.resource_type,
    a.resource_name,
    a.action_type,
    a.status,
    a.executed_at,
    a.duration_ms,
    a.error_message,
    a.executed_by,
    a.source,
    r.estimated_savings_usd as expected_savings,
    a.created_at
FROM actions a
LEFT JOIN recommendations r ON a.recommendation_id = r.id
ORDER BY a.executed_at DESC NULLS LAST, a.created_at DESC
LIMIT 100;

-- View for action success rate analysis
CREATE OR REPLACE VIEW action_success_rate_view AS
SELECT 
    action_type,
    status,
    COUNT(*) as count,
    AVG(duration_ms) as avg_duration_ms,
    COUNT(*) FILTER (WHERE status = 'success') as success_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count
FROM actions
WHERE executed_at >= NOW() - INTERVAL '30 days'
GROUP BY action_type, status
ORDER BY action_type, status;

-- View for failed actions requiring attention
CREATE OR REPLACE VIEW failed_actions_view AS
SELECT 
    a.id,
    a.resource_id,
    a.resource_type,
    a.action_type,
    a.status,
    a.error_message,
    a.error_code,
    a.executed_at,
    a.created_at,
    r.title as recommendation_title,
    r.estimated_savings_usd
FROM actions a
LEFT JOIN recommendations r ON a.recommendation_id = r.id
WHERE a.status = 'failed'
  AND a.created_at >= NOW() - INTERVAL '7 days'
ORDER BY a.created_at DESC;

-- Table comments for documentation
COMMENT ON TABLE actions IS 'Tracks all executed AWS resource actions for audit and monitoring';
COMMENT ON COLUMN actions.resource_id IS 'AWS resource identifier (instance ID, volume ID, etc.)';
COMMENT ON COLUMN actions.action_type IS 'Type of action: stop_ec2, delete_ebs, resize_rds, etc.';
COMMENT ON COLUMN actions.status IS 'Current status: pending, in_progress, success, failed, cancelled, timeout';
COMMENT ON COLUMN actions.executed_at IS 'When the action was started';
COMMENT ON COLUMN actions.completed_at IS 'When the action finished (success or failure)';
COMMENT ON COLUMN actions.error_message IS 'Detailed error message if action failed';
COMMENT ON COLUMN actions.error_code IS 'Categorized error code for analysis';
COMMENT ON COLUMN actions.request_params IS 'JSON parameters sent to AWS API';
COMMENT ON COLUMN actions.response_data IS 'JSON response from AWS API';
COMMENT ON COLUMN actions.recommendation_id IS 'Reference to the recommendation that triggered this action';
COMMENT ON COLUMN actions.executed_by IS 'User or service that initiated the action';
COMMENT ON COLUMN actions.source IS 'Source of action: api, scheduler, manual';
COMMENT ON COLUMN actions.duration_ms IS 'Action execution duration in milliseconds';
