-- Create resources table for storing cloud resource information
-- This table supports multi-cloud resource discovery and cost optimization

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create resources table
CREATE TABLE IF NOT EXISTS resources (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Cloud resource identification
    resource_id VARCHAR(255) NOT NULL,  -- Cloud provider resource ID (e.g., i-1234567890abcdef0)
    resource_type VARCHAR(50) NOT NULL,  -- Resource type (EC2, RDS, EBS, etc.)
    provider VARCHAR(50) NOT NULL,       -- Cloud provider (aws, gcp, azure)
    region VARCHAR(50) NOT NULL,         -- Cloud region
    account_id VARCHAR(50) NOT NULL,    -- Cloud account ID
    
    -- Resource information
    name VARCHAR(255) NOT NULL,          -- Resource name (from tags or resource ID)
    state VARCHAR(50) NOT NULL,         -- Resource state (running, stopped, available, etc.)
    instance_type VARCHAR(100),         -- Instance type (t3.micro, db.t3.micro, etc.)
    
    -- Flexible storage for tags and metadata
    tags JSONB NOT NULL DEFAULT '{}',   -- Resource tags as JSONB
    metadata JSONB NOT NULL DEFAULT '{}', -- Additional metadata as JSONB
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT resources_resource_id_unique UNIQUE (resource_id),
    CONSTRAINT resources_resource_type_check CHECK (resource_type IN ('EC2', 'RDS', 'EBS', 'Lambda', 'S3', 'VPC', 'Subnet', 'EKS', 'IAM', 'CloudFront', 'ELB', 'ALB', 'NLB', 'AutoScaling', 'CloudWatch', 'SNS', 'SQS', 'Route53', 'SecretsManager', 'KMS', 'VPCFlowLogs', 'Config', 'CloudTrail', 'GuardDuty', 'SecurityHub', 'Inspector', 'Macie', 'Backup', 'StorageGateway', 'DirectConnect', 'VPN', 'TransitGateway', 'NATGateway', 'InternetGateway', 'ElasticIP', 'LoadBalancer', 'TargetGroup', 'AutoScalingGroup', 'LaunchConfiguration', 'LaunchTemplate', 'SecurityGroup', 'NetworkACL', 'RouteTable', 'Subnet', 'VPC', 'EFS', 'FSx', 'ECS', 'EKS', 'Fargate', 'Lambda', 'StepFunctions', 'EventBridge', 'SQS', 'SNS', 'Kinesis', 'DynamoDB', 'ElastiCache', 'Neptune', 'Redshift', 'Aurora', 'RDS', 'DocumentDB', 'Keyspaces', 'Timestream', 'OpenSearch', 'ElasticSearch', 'CloudFront', 'Route53', 'CertificateManager', 'SecretsManager', 'KMS', 'CloudHSM', 'IAM', 'Organizations', 'ControlTower', 'ServiceCatalog', 'Config', 'CloudTrail', 'CloudWatch', 'XRay', 'Inspector', 'Macie', 'GuardDuty', 'SecurityHub', 'ComputeOptimizer', 'CostExplorer', 'Budgets', 'CUR', 'WellArchitected', 'TrustedAdvisor', 'Support', 'Marketplace', 'CodePipeline', 'CodeBuild', 'CodeDeploy', 'CodeCommit', 'CloudFormation', 'CDK', 'SAM', 'Terraform', 'Pulumi', 'Ansible', 'Chef', 'Puppet', 'Kubernetes', 'Docker', 'Jenkins', 'GitHub', 'GitLab', 'Bitbucket', 'Jira', 'Confluence', 'Slack', 'Teams', 'Zoom', 'Office365', 'GoogleWorkspace', 'Salesforce', 'Zendesk', 'Datadog', 'NewRelic', 'PagerDuty', 'Splunk', 'ELK', 'Grafana', 'Prometheus', 'Kubernetes', 'Istio', 'Linkerd', 'Consul', 'Vault', 'Nomad', 'Packer', 'Vagrant', 'AnsibleTower', 'Rundeck', 'Jenkins', 'GitLabCI', 'GitHubActions', 'CircleCI', 'TravisCI', 'AppVeyor', 'CodeShip', 'Wercker', 'Drone', 'Semaphore', 'Bamboo', 'TeamCity', 'OctopusDeploy', 'Spinnaker', 'ArgoCD', 'Flux', 'Helm', 'Kustomize', 'Skaffold', 'Tilt', 'Telepresence', 'K9s', 'Lens', 'Octant', 'Kubeflow', 'Airflow', 'Prefect', 'Dagster', 'Luigi', 'Azkaban', 'Oozie', 'NiFi', 'Kafka', 'Pulsar', 'RabbitMQ', 'ActiveMQ', 'Redis', 'Memcached', 'Cassandra', 'ScyllaDB', 'CockroachDB', 'TiDB', 'YugabyteDB', 'FoundationDB', 'BadgerDB', 'BoltDB', 'LevelDB', 'RocksDB', 'LMDB', 'SurrealDB', 'EdgeDB', 'FaunaDB', 'PlanetScale', 'Neon', 'Supabase', 'Turso', 'Upstash', 'RedisLabs', 'Momento', 'DynamoDB', 'Cassandra', 'ScyllaDB', 'CockroachDB', 'TiDB', 'YugabyteDB', 'FoundationDB', 'BadgerDB', 'BoltDB', 'LevelDB', 'RocksDB', 'LMDB', 'SurrealDB', 'EdgeDB', 'FaunaDB', 'PlanetScale', 'Neon', 'Supabase', 'Turso', 'Upstash', 'RedisLabs', 'Momento')),
    CONSTRAINT resources_provider_check CHECK (provider IN ('aws', 'gcp', 'azure', 'digitalocean', 'linode', 'vultr', 'upcloud', 'scaleway', 'ovh', 'hetzner', 'ibm', 'oracle', 'alibaba', 'tencent', 'baidu', 'huawei', 'google', 'microsoft', 'amazon', 'facebook', 'apple', 'netflix', 'spotify', 'uber', 'airbnb', 'linkedin', 'twitter', 'instagram', 'youtube', 'tiktok', 'snapchat', 'reddit', 'discord', 'slack', 'microsoft', 'google', 'amazon', 'apple', 'facebook', 'netflix', 'spotify', 'uber', 'airbnb', 'linkedin', 'twitter', 'instagram', 'youtube', 'tiktok', 'snapchat', 'reddit', 'discord', 'slack')),
    CONSTRAINT resources_state_check CHECK (state IN ('pending', 'running', 'stopping', 'stopped', 'shutting-down', 'terminated', 'rebooting', 'available', 'creating', 'modifying', 'deleting', 'backup-restoring', 'in-use', 'detaching', 'attaching', 'detached', 'attached', 'busy', 'impaired', 'optimizing', 'suspended', 'resuming', 'archived', 'converting', 'failed', 'inaccessible', 'storage-optimization', 'migrating', 'pending-acceptance', 'pending-verification', 'verification-failed', 'pending-restore', 'copying', 'restoring', 'recycling', 'retaining', 'pending-deletion', 'deleting', 'deleted', 'available', 'backing-up', 'modifying', 'resetting-master-credentials', 'upgrading', 'configuring-enhanced-monitoring', 'configuring-iam-database-auth', 'moving', 'updating', 'rebooting', 'resetting', 'starting', 'stopping', 'maintenance', 'refreshing', 'snapshotting', 'upgrading', 'copying', 'testing', 'syncing', 'preparing', 'importing', 'exporting', 'recovering', 'purging', 'seeding', 'initializing', 'finalizing', 'committing', 'rolling-back', 'rolling-forward', 'validating', 'authorizing', 'synchronizing', 'desynchronizing', 'suspending', 'resuming', 'activating', 'deactivating', 'enabling', 'disabling', 'configuring', 'reconfiguring', 'rebuilding', 'reindexing', 'optimizing', 'compacting', 'vacuuming', 'analyzing', 'checking', 'repairing', 'recovering', 'restoring', 'backing-up', 'restoring-from-backup', 'creating-backup', 'deleting-backup', 'listing-backups', 'describing-backups', 'sharing-backup', 'unsharing-backup', 'copying-backup', 'moving-backup', 'renaming-backup', 'exporting-backup', 'importing-backup', 'scheduling-backup', 'canceling-backup', 'pausing-backup', 'resuming-backup', 'modifying-backup', 'tagging-backup', 'untagging-backup', 'describing-backup-jobs', 'listing-backup-jobs', 'canceling-backup-job', 'starting-backup-job', 'stopping-backup-job', 'restarting-backup-job', 'completing-backup-job', 'failing-backup-job', 'succeeding-backup-job', 'pending-backup-job', 'running-backup-job', 'completed-backup-job', 'failed-backup-job', 'cancelled-backup-job', 'expired-backup-job', 'available-backup-job', 'unavailable-backup-job', 'creating-backup-job', 'deleting-backup-job', 'modifying-backup-job', 'tagging-backup-job', 'untagging-backup-job', 'describing-backup-job', 'listing-backup-jobs', 'canceling-backup-job', 'starting-backup-job', 'stopping-backup-job', 'restarting-backup-job', 'completing-backup-job', 'failing-backup-job', 'succeeding-backup-job', 'pending-backup-job', 'running-backup-job', 'completed-backup-job', 'failed-backup-job', 'cancelled-backup-job', 'expired-backup-job', 'available-backup-job', 'unavailable-backup-job')),
    CONSTRAINT resources_created_at_check CHECK (created_at <= updated_at)
);

-- Create indexes for performance

-- Primary index on resource_id for upsert operations
CREATE INDEX IF NOT EXISTS idx_resources_resource_id ON resources(resource_id);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_resources_resource_type ON resources(resource_type);
CREATE INDEX IF NOT EXISTS idx_resources_provider ON resources(provider);
CREATE INDEX IF NOT EXISTS idx_resources_region ON resources(region);
CREATE INDEX IF NOT EXISTS idx_resources_account_id ON resources(account_id);
CREATE INDEX IF NOT EXISTS idx_resources_state ON resources(state);
CREATE INDEX IF NOT EXISTS idx_resources_instance_type ON resources(instance_type);

-- GIN index on tags for JSONB queries
CREATE INDEX IF NOT EXISTS idx_resources_tags_gin ON resources USING GIN(tags);

-- GIN index on metadata for JSONB queries
CREATE INDEX IF NOT EXISTS idx_resources_metadata_gin ON resources USING GIN(metadata);

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_resources_provider_type ON resources(provider, resource_type);
CREATE INDEX IF NOT EXISTS idx_resources_region_type ON resources(region, resource_type);
CREATE INDEX IF NOT EXISTS idx_resources_account_type ON resources(account_id, resource_type);
CREATE INDEX IF NOT EXISTS idx_resources_state_type ON resources(state, resource_type);

-- Time-based indexes for analytics
CREATE INDEX IF NOT EXISTS idx_resources_created_at ON resources(created_at);
CREATE INDEX IF NOT EXISTS idx_resources_updated_at ON resources(updated_at);

-- Partial indexes for common filtered queries
CREATE INDEX IF NOT EXISTS idx_resources_active_ec2 ON resources(resource_type, state) 
WHERE resource_type = 'EC2' AND state IN ('running', 'pending', 'stopping', 'rebooting');

CREATE INDEX IF NOT EXISTS idx_resources_active_rds ON resources(resource_type, state) 
WHERE resource_type = 'RDS' AND state IN ('available', 'creating', 'modifying', 'rebooting', 'resetting');

CREATE INDEX IF NOT EXISTS idx_resources_active_ebs ON resources(resource_type, state) 
WHERE resource_type = 'EBS' AND state IN ('available', 'in-use', 'modifying', 'creating');

-- Partial indexes for production resources
CREATE INDEX IF NOT EXISTS idx_resources_production ON resources(resource_type, created_at) 
WHERE tags->>'Environment' = 'production';

-- Partial indexes for cost optimization
CREATE INDEX IF NOT EXISTS idx_resources_optimization_candidates ON resources(resource_type, state, updated_at) 
WHERE resource_type IN ('EC2', 'RDS', 'EBS') 
  AND state IN ('running', 'available', 'in-use')
  AND tags->>'Environment' != 'production';

-- Create trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_resources_updated_at 
    BEFORE UPDATE ON resources 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE resources IS 'Cloud resource information for cost optimization and management';

COMMENT ON COLUMN resources.id IS 'Primary key - UUID for global uniqueness';
COMMENT ON COLUMN resources.resource_id IS 'Cloud provider resource identifier (e.g., i-1234567890abcdef0, vol-1234567890abcdef0)';
COMMENT ON COLUMN resources.resource_type IS 'Type of cloud resource (EC2, RDS, EBS, Lambda, S3, etc.)';
COMMENT ON COLUMN resources.provider IS 'Cloud provider (aws, gcp, azure, digitalocean, etc.)';
COMMENT ON COLUMN resources.region IS 'Cloud region where resource is deployed';
COMMENT ON COLUMN resources.account_id IS 'Cloud account ID that owns the resource';
COMMENT ON COLUMN resources.name IS 'Human-readable name (from tags or derived from resource ID)';
COMMENT ON COLUMN resources.state IS 'Current state of the resource (running, stopped, available, etc.)';
COMMENT ON COLUMN resources.instance_type IS 'Instance type or size (t3.micro, db.t3.micro, gp3, etc.)';
COMMENT ON COLUMN resources.tags IS 'Resource tags stored as JSONB for flexible querying';
COMMENT ON COLUMN resources.metadata IS 'Additional metadata stored as JSONB for flexible schema';
COMMENT ON COLUMN resources.created_at IS 'When the resource was first discovered/created';
COMMENT ON COLUMN resources.updated_at IS 'When the resource was last updated';

-- Create view for resource summary statistics
CREATE OR REPLACE VIEW resource_summary AS
SELECT 
    provider,
    region,
    resource_type,
    state,
    COUNT(*) as resource_count,
    COUNT(DISTINCT account_id) as account_count,
    MIN(created_at) as oldest_resource,
    MAX(created_at) as newest_resource,
    MAX(updated_at) as last_updated
FROM resources
GROUP BY provider, region, resource_type, state
ORDER BY provider, region, resource_type, state;

COMMENT ON VIEW resource_summary IS 'Summary statistics of resources by provider, region, type, and state';

-- Create view for resource cost optimization candidates
CREATE OR REPLACE VIEW optimization_candidates AS
SELECT 
    id,
    resource_id,
    resource_type,
    provider,
    region,
    account_id,
    name,
    state,
    instance_type,
    tags,
    metadata,
    created_at,
    updated_at,
    -- Optimization score based on various factors
    CASE 
        WHEN tags->>'Environment' = 'production' THEN 0
        WHEN state IN ('running', 'available', 'in-use') THEN 1
        WHEN state IN ('stopped', 'available') THEN 2
        ELSE 3
    END as optimization_priority
FROM resources
WHERE resource_type IN ('EC2', 'RDS', 'EBS')
  AND state IN ('running', 'available', 'in-use', 'stopped')
  AND tags->>'Environment' != 'production'
ORDER BY optimization_priority, updated_at;

COMMENT ON VIEW optimization_candidates IS 'Resources that are candidates for cost optimization';

-- Create function to search resources by tags
CREATE OR REPLACE FUNCTION search_resources_by_tag(tag_key TEXT, tag_value TEXT)
RETURNS TABLE (
    id UUID,
    resource_id VARCHAR(255),
    resource_type VARCHAR(50),
    provider VARCHAR(50),
    region VARCHAR(50),
    account_id VARCHAR(50),
    name VARCHAR(255),
    state VARCHAR(50),
    instance_type VARCHAR(100),
    tags JSONB,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        r.id,
        r.resource_id,
        r.resource_type,
        r.provider,
        r.region,
        r.account_id,
        r.name,
        r.state,
        r.instance_type,
        r.tags,
        r.metadata,
        r.created_at,
        r.updated_at
    FROM resources r
    WHERE r.tags->>tag_key = tag_value
    ORDER BY r.updated_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Create function to get resource statistics
CREATE OR REPLACE FUNCTION get_resource_statistics()
RETURNS TABLE (
    total_resources BIGINT,
    resources_by_type JSONB,
    resources_by_provider JSONB,
    resources_by_region JSONB,
    resources_by_state JSONB,
    resources_by_account JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        (SELECT COUNT(*) FROM resources) as total_resources,
        (SELECT jsonb_object_agg(resource_type, count) 
         FROM (SELECT resource_type, COUNT(*) as count FROM resources GROUP BY resource_type) t) as resources_by_type,
        (SELECT jsonb_object_agg(provider, count) 
         FROM (SELECT provider, COUNT(*) as count FROM resources GROUP BY provider) t) as resources_by_provider,
        (SELECT jsonb_object_agg(region, count) 
         FROM (SELECT region, COUNT(*) as count FROM resources GROUP BY region) t) as resources_by_region,
        (SELECT jsonb_object_agg(state, count) 
         FROM (SELECT state, COUNT(*) as count FROM resources GROUP BY state) t) as resources_by_state,
        (SELECT jsonb_object_agg(account_id, count) 
         FROM (SELECT account_id, COUNT(*) as count FROM resources GROUP BY account_id) t) as resources_by_account;
END;
$$ LANGUAGE plpgsql;
