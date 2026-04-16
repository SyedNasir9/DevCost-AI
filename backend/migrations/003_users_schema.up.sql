-- DevCost AI Users Schema Migration
-- Migration: 003_users_schema
-- Created: 2026-03-26
-- Description: Create user management and cloud account tables

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) DEFAULT 'user',
    is_active BOOLEAN DEFAULT true,
    email_verified BOOLEAN DEFAULT false,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT users_role_check CHECK (role IN ('admin', 'user', 'viewer')),
    CONSTRAINT users_email_check CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

-- Cloud accounts table - links users to their cloud provider accounts
CREATE TABLE IF NOT EXISTS cloud_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    credentials_encrypted TEXT, -- Encrypted cloud provider credentials
    is_active BOOLEAN DEFAULT true,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    sync_status VARCHAR(20) DEFAULT 'pending', -- pending, success, error
    sync_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT cloud_accounts_provider_check CHECK (provider IN ('aws', 'gcp', 'azure')),
    CONSTRAINT cloud_accounts_sync_status_check CHECK (
        sync_status IN ('pending', 'success', 'error', 'syncing')
    ),
    CONSTRAINT cloud_accounts_unique_account UNIQUE(provider, account_id)
);

-- Cost alerts table
CREATE TABLE IF NOT EXISTS cost_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cloud_account_id UUID REFERENCES cloud_accounts(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- budget, anomaly, threshold, etc.
    threshold_amount DECIMAL(12,4),
    current_amount DECIMAL(12,4),
    currency VARCHAR(3) DEFAULT 'USD',
    condition_operator VARCHAR(10) DEFAULT 'gt', -- gt, lt, gte, lte
    notification_channels JSONB, -- email, slack, webhook, etc.
    is_active BOOLEAN DEFAULT true,
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT cost_alerts_type_check CHECK (
        type IN ('budget', 'anomaly', 'threshold', 'trend', 'daily_limit')
    ),
    CONSTRAINT cost_alerts_operator_check CHECK (
        condition_operator IN ('gt', 'lt', 'gte', 'lte', 'eq')
    ),
    CONSTRAINT cost_alerts_threshold_check CHECK (threshold_amount > 0)
);

-- Alert notifications table - tracks sent notifications
CREATE TABLE IF NOT EXISTS alert_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alert_id UUID NOT NULL REFERENCES cost_alerts(id) ON DELETE CASCADE,
    channel VARCHAR(50) NOT NULL, -- email, slack, webhook
    status VARCHAR(20) DEFAULT 'pending', -- pending, sent, failed
    message TEXT,
    error_details TEXT,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT alert_notifications_channel_check CHECK (
        channel IN ('email', 'slack', 'webhook', 'sms', 'push')
    ),
    CONSTRAINT alert_notifications_status_check CHECK (
        status IN ('pending', 'sent', 'failed', 'retry')
    )
);

-- Indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login_at);

-- Indexes for cloud_accounts table
CREATE INDEX IF NOT EXISTS idx_cloud_accounts_user_id ON cloud_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_cloud_accounts_provider ON cloud_accounts(provider);
CREATE INDEX IF NOT EXISTS idx_cloud_accounts_is_active ON cloud_accounts(is_active);
CREATE INDEX IF NOT EXISTS idx_cloud_accounts_last_sync ON cloud_accounts(last_sync_at);
CREATE INDEX IF NOT EXISTS idx_cloud_accounts_sync_status ON cloud_accounts(sync_status);

-- Indexes for cost_alerts table
CREATE INDEX IF NOT EXISTS idx_cost_alerts_user_id ON cost_alerts(user_id);
CREATE INDEX IF NOT EXISTS idx_cost_alerts_cloud_account_id ON cost_alerts(cloud_account_id);
CREATE INDEX IF NOT EXISTS idx_cost_alerts_type ON cost_alerts(type);
CREATE INDEX IF NOT EXISTS idx_cost_alerts_is_active ON cost_alerts(is_active);
CREATE INDEX IF NOT EXISTS idx_cost_alerts_last_triggered ON cost_alerts(last_triggered_at);

-- Indexes for alert_notifications table
CREATE INDEX IF NOT EXISTS idx_alert_notifications_alert_id ON alert_notifications(alert_id);
CREATE INDEX IF NOT EXISTS idx_alert_notifications_channel ON alert_notifications(channel);
CREATE INDEX IF NOT EXISTS idx_alert_notifications_status ON alert_notifications(status);
CREATE INDEX IF NOT EXISTS idx_alert_notifications_sent_at ON alert_notifications(sent_at);

-- Table comments
COMMENT ON TABLE users IS 'User accounts for DevCost AI platform';
COMMENT ON TABLE cloud_accounts IS 'Cloud provider accounts linked to users';
COMMENT ON TABLE cost_alerts IS 'Cost monitoring and alerting rules';
COMMENT ON TABLE alert_notifications IS 'History of alert notifications sent';

-- Column comments
COMMENT ON COLUMN users.password_hash IS 'bcrypt hash of user password';
COMMENT ON COLUMN users.email_verified IS 'Whether user email has been verified';
COMMENT ON COLUMN cloud_accounts.credentials_encrypted IS 'Encrypted cloud provider credentials';
COMMENT ON COLUMN cloud_accounts.sync_status IS 'Status of last cloud account sync';
COMMENT ON COLUMN cost_alerts.condition_operator IS 'Comparison operator for alert conditions';
COMMENT ON COLUMN cost_alerts.notification_channels IS 'JSON array of notification channels';
