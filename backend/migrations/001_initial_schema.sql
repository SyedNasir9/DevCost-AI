-- DevCost AI Initial Database Schema
-- Created: 2026-03-26

-- Users table for authentication and management
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) DEFAULT 'user',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Cloud accounts for different providers
CREATE TABLE IF NOT EXISTS cloud_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL, -- 'aws', 'gcp', 'azure'
    account_name VARCHAR(255) NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    credentials_encrypted TEXT, -- encrypted credentials
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(provider, account_id)
);

-- Cost data storage
CREATE TABLE IF NOT EXISTS cost_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cloud_account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,
    service_name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(255),
    cost_amount DECIMAL(12, 4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    usage_quantity DECIMAL(12, 4),
    usage_unit VARCHAR(100),
    billing_period_start DATE NOT NULL,
    billing_period_end DATE NOT NULL,
    tags JSONB, -- flexible storage for cloud provider tags
    raw_data JSONB, -- original response from cloud provider
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Cost optimization recommendations
CREATE TABLE IF NOT EXISTS recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cloud_account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,
    recommendation_type VARCHAR(100) NOT NULL, -- 'right_sizing', 'reserved_instances', etc.
    resource_id VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255),
    current_cost DECIMAL(12, 4),
    potential_savings DECIMAL(12, 4),
    confidence_score DECIMAL(3, 2), -- 0.00 to 1.00
    status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'implemented', 'dismissed'
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    implemented_at TIMESTAMP WITH TIME ZONE
);

-- Cost alerts and notifications
CREATE TABLE IF NOT EXISTS cost_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cloud_account_id UUID REFERENCES cloud_accounts(id) ON DELETE CASCADE,
    alert_type VARCHAR(100) NOT NULL, -- 'budget_exceeded', 'anomaly_detected', etc.
    threshold_amount DECIMAL(12, 4),
    current_amount DECIMAL(12, 4),
    message TEXT NOT NULL,
    is_read BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API keys for external integrations
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    permissions JSONB, -- list of allowed endpoints/actions
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_cloud_accounts_user_id ON cloud_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_cost_data_cloud_account_id ON cost_data(cloud_account_id);
CREATE INDEX IF NOT EXISTS idx_cost_data_billing_period ON cost_data(billing_period_start, billing_period_end);
CREATE INDEX IF NOT EXISTS idx_recommendations_cloud_account_id ON recommendations(cloud_account_id);
CREATE INDEX IF NOT EXISTS idx_cost_alerts_user_id ON cost_alerts(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
