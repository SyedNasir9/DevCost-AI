-- Migration: 006_recommendations_schema.up.sql
-- Creates the recommendations table for storing resource optimization recommendations

-- Create recommendations table
CREATE TABLE IF NOT EXISTS recommendations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Resource identification
    resource_id VARCHAR(255) NOT NULL,
    resource_uuid UUID REFERENCES resources(id) ON DELETE CASCADE,
    resource_type VARCHAR(50) NOT NULL,
    resource_name VARCHAR(255),
    
    -- Recommendation details
    recommendation_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    priority VARCHAR(50) NOT NULL DEFAULT 'medium',
    
    -- Content
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    rationale TEXT,
    
    -- State information (stored as JSONB for flexibility)
    current_state JSONB DEFAULT '{}',
    proposed_state JSONB DEFAULT '{}',
    
    -- Financial impact
    estimated_savings_usd DECIMAL(12, 2) NOT NULL DEFAULT 0.0,
    savings_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    risk_level VARCHAR(20) NOT NULL DEFAULT 'low',
    
    -- Implementation guidance
    implementation_steps JSONB DEFAULT '[]',
    alternatives JSONB DEFAULT '[]',
    
    -- Source references (waste_results FK added in 009_add_waste_fk.up.sql)
    waste_id UUID,
    cost_data_id UUID REFERENCES cost_data(id) ON DELETE SET NULL,
    
    -- Validity period
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMP WITH TIME ZONE,
    
    -- Implementation tracking
    implemented_at TIMESTAMP WITH TIME ZONE,
    implemented_by VARCHAR(255),
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT recommendations_type_check CHECK (recommendation_type IN ('stop', 'delete', 'resize', 'schedule', 'snapshot')),
    CONSTRAINT recommendations_status_check CHECK (status IN ('active', 'pending', 'accepted', 'rejected', 'implemented', 'expired')),
    CONSTRAINT recommendations_priority_check CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    CONSTRAINT recommendations_savings_positive CHECK (estimated_savings_usd >= 0),
    CONSTRAINT recommendations_risk_check CHECK (risk_level IN ('low', 'medium', 'high', 'critical'))
);

-- Indexes for efficient querying

-- Primary lookup indexes
CREATE INDEX IF NOT EXISTS idx_recommendations_resource_id ON recommendations(resource_id);
CREATE INDEX IF NOT EXISTS idx_recommendations_resource_uuid ON recommendations(resource_uuid);

-- Status and priority indexes
CREATE INDEX IF NOT EXISTS idx_recommendations_status ON recommendations(status);
CREATE INDEX IF NOT EXISTS idx_recommendations_priority ON recommendations(priority);

-- Composite index for active high-priority recommendations
CREATE INDEX IF NOT EXISTS idx_recommendations_active_priority 
ON recommendations(status, priority) 
WHERE status = 'active';

-- Index for resource type filtering
CREATE INDEX IF NOT EXISTS idx_recommendations_resource_type ON recommendations(resource_type);

-- Index for recommendation type
CREATE INDEX IF NOT EXISTS idx_recommendations_type ON recommendations(recommendation_type);

-- Index for savings amount (for sorting by impact)
CREATE INDEX IF NOT EXISTS idx_recommendations_savings ON recommendations(estimated_savings_usd DESC);

-- Index for validity period
CREATE INDEX IF NOT EXISTS idx_recommendations_validity ON recommendations(valid_from, valid_until);

-- Partial index for unimplemented recommendations
CREATE INDEX IF NOT EXISTS idx_recommendations_unimplemented 
ON recommendations(resource_id, created_at) 
WHERE status NOT IN ('implemented', 'rejected');

-- Index for waste_id lookup
CREATE INDEX IF NOT EXISTS idx_recommendations_waste_id ON recommendations(waste_id);

-- Index for cost_data_id lookup
CREATE INDEX IF NOT EXISTS idx_recommendations_cost_data_id ON recommendations(cost_data_id);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_recommendations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for updated_at
CREATE TRIGGER recommendations_updated_at_trigger
    BEFORE UPDATE ON recommendations
    FOR EACH ROW
    EXECUTE FUNCTION update_recommendations_updated_at();

-- View for active recommendations with resource details
CREATE OR REPLACE VIEW active_recommendations_view AS
SELECT 
    r.id,
    r.resource_id,
    r.resource_type,
    r.resource_name,
    r.recommendation_type,
    r.priority,
    r.title,
    r.description,
    r.estimated_savings_usd,
    r.risk_level,
    res.region,
    res.account_id,
    res.provider,
    r.valid_from,
    r.valid_until,
    r.created_at
FROM recommendations r
LEFT JOIN resources res ON r.resource_uuid = res.id
WHERE r.status = 'active'
ORDER BY r.estimated_savings_usd DESC;

-- View for recommendations summary by resource type
CREATE OR REPLACE VIEW recommendations_summary_by_type AS
SELECT 
    resource_type,
    recommendation_type,
    priority,
    COUNT(*) as count,
    SUM(estimated_savings_usd) as total_estimated_savings,
    AVG(estimated_savings_usd) as avg_estimated_savings,
    MIN(estimated_savings_usd) as min_estimated_savings,
    MAX(estimated_savings_usd) as max_estimated_savings
FROM recommendations
WHERE status = 'active'
GROUP BY resource_type, recommendation_type, priority
ORDER BY total_estimated_savings DESC;

-- View for high impact recommendations
CREATE OR REPLACE VIEW high_impact_recommendations AS
SELECT 
    id,
    resource_id,
    resource_type,
    resource_name,
    recommendation_type,
    title,
    estimated_savings_usd,
    priority,
    risk_level,
    valid_from
FROM recommendations
WHERE status = 'active'
  AND (estimated_savings_usd > 100.0 OR priority IN ('critical', 'high'))
ORDER BY estimated_savings_usd DESC;

-- Function to expire old recommendations
CREATE OR REPLACE FUNCTION expire_old_recommendations(days_old INT DEFAULT 30)
RETURNS INT AS $$
DECLARE
    updated_count INT;
BEGIN
    UPDATE recommendations
    SET status = 'expired',
        valid_until = NOW()
    WHERE status = 'active'
      AND created_at < NOW() - INTERVAL '1 day' * days_old;
    
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RETURN updated_count;
END;
$$ LANGUAGE plpgsql;

-- Materialized view for recommendation analytics
CREATE MATERIALIZED VIEW recommendation_analytics AS
SELECT 
    DATE_TRUNC('month', created_at) as month,
    resource_type,
    recommendation_type,
    COUNT(*) as recommendations_generated,
    COUNT(*) FILTER (WHERE status = 'implemented') as recommendations_implemented,
    SUM(estimated_savings_usd) as total_potential_savings,
    SUM(estimated_savings_usd) FILTER (WHERE status = 'implemented') as total_realized_savings,
    AVG(estimated_savings_usd) as avg_potential_savings
FROM recommendations
GROUP BY DATE_TRUNC('month', created_at), resource_type, recommendation_type
ORDER BY month DESC, total_potential_savings DESC;

-- Index on materialized view
CREATE INDEX idx_recommendation_analytics_month ON recommendation_analytics(month);
CREATE INDEX idx_recommendation_analytics_type ON recommendation_analytics(resource_type);

-- Function to refresh analytics
CREATE OR REPLACE FUNCTION refresh_recommendation_analytics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY recommendation_analytics;
END;
$$ LANGUAGE plpgsql;

-- Table comments for documentation
COMMENT ON TABLE recommendations IS 'Stores resource optimization recommendations generated from waste detection and cost analysis';
COMMENT ON COLUMN recommendations.resource_id IS 'AWS resource identifier';
COMMENT ON COLUMN recommendations.recommendation_type IS 'Type of action: stop, delete, resize, schedule, snapshot';
COMMENT ON COLUMN recommendations.status IS 'Current status: active, pending, accepted, rejected, implemented, expired';
COMMENT ON COLUMN recommendations.priority IS 'Priority level: critical, high, medium, low';
COMMENT ON COLUMN recommendations.estimated_savings_usd IS 'Estimated monthly savings in USD';
COMMENT ON COLUMN recommendations.risk_level IS 'Risk level of implementation: low, medium, high, critical';
COMMENT ON COLUMN recommendations.implementation_steps IS 'JSON array of implementation steps';
COMMENT ON COLUMN recommendations.waste_id IS 'Reference to waste detection result that triggered this recommendation';
COMMENT ON COLUMN recommendations.cost_data_id IS 'Reference to cost data used for savings calculation';
