-- Migration: 008_waste_results_schema.up.sql
-- Creates the waste_results table for storing detected waste resources

-- Create waste_results table
CREATE TABLE IF NOT EXISTS waste_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Resource identification
    resource_id VARCHAR(255) NOT NULL,
    resource_uuid UUID REFERENCES resources(id) ON DELETE CASCADE,
    resource_type VARCHAR(50) NOT NULL,
    resource_name VARCHAR(255),
    
    -- Waste detection details
    waste_type VARCHAR(50) NOT NULL,
    reason TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'medium',
    
    -- Financial impact
    estimated_savings_usd DECIMAL(12, 2) NOT NULL DEFAULT 0.0,
    confidence DECIMAL(3, 2) NOT NULL DEFAULT 0.0,
    
    -- Detection metadata
    detection_criteria JSONB DEFAULT '{}',
    metrics JSONB DEFAULT '{}',
    
    -- Timestamps
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT waste_results_type_check CHECK (waste_type IN ('idle_ec2', 'unattached_ebs', 'underutilized_rds', 'oversized_instance', 'unused_elastic_ip', 'empty_bucket', 'other')),
    CONSTRAINT waste_results_severity_check CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT waste_results_savings_positive CHECK (estimated_savings_usd >= 0),
    CONSTRAINT waste_results_confidence_range CHECK (confidence >= 0 AND confidence <= 1)
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_waste_results_resource_id ON waste_results(resource_id);
CREATE INDEX IF NOT EXISTS idx_waste_results_resource_uuid ON waste_results(resource_uuid);
CREATE INDEX IF NOT EXISTS idx_waste_results_waste_type ON waste_results(waste_type);
CREATE INDEX IF NOT EXISTS idx_waste_results_severity ON waste_results(severity);
CREATE INDEX IF NOT EXISTS idx_waste_results_detected_at ON waste_results(detected_at);

-- Partial index for unresolved waste
CREATE INDEX IF NOT EXISTS idx_waste_results_unresolved 
ON waste_results(resource_id, detected_at) 
WHERE resolved_at IS NULL;

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_waste_results_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for updated_at
CREATE TRIGGER waste_results_updated_at_trigger
    BEFORE UPDATE ON waste_results
    FOR EACH ROW
    EXECUTE FUNCTION update_waste_results_updated_at();

-- View for active waste
CREATE OR REPLACE VIEW active_waste_view AS
SELECT 
    w.id,
    w.resource_id,
    w.resource_type,
    w.resource_name,
    w.waste_type,
    w.reason,
    w.severity,
    w.estimated_savings_usd,
    w.confidence,
    w.detected_at
FROM waste_results w
WHERE w.resolved_at IS NULL
ORDER BY w.estimated_savings_usd DESC;

-- Table comment
COMMENT ON TABLE waste_results IS 'Stores detected resource waste for cost optimization';
