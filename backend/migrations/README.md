# DevCost AI Database Migrations

This directory contains SQL migration files for the DevCost AI cloud cost optimization platform.

## Migration Files

### 001_initial_schema.sql
- **Status**: Legacy - Replaced by newer migrations
- **Description**: Initial basic schema (deprecated)
- **Note**: Use migrations 002+ instead

### 002_cloud_resources_schema.up.sql
- **Version**: 002
- **Description**: Core cloud resources schema
- **Tables Created**:
  - `resources` - Cloud resource inventory
  - `resource_usage` - Resource utilization metrics
  - `cost_data` - Cost tracking and billing data
  - `recommendations` - Cost optimization recommendations
  - `actions` - Resource action tracking

### 003_users_schema.up.sql
- **Version**: 003
- **Description**: User management and cloud accounts
- **Tables Created**:
  - `users` - User authentication and management
  - `cloud_accounts` - Cloud provider account connections
  - `cost_alerts` - Cost monitoring and alerting
  - `alert_notifications` - Alert notification history

## Schema Design Overview

### Core Tables

#### Resources
- **resources**: Master table for all cloud resources
  - Supports multi-cloud (AWS, GCP, Azure)
  - Flexible tagging with JSONB
  - Resource state and metadata tracking

#### Usage & Cost Tracking
- **resource_usage**: Time-series utilization metrics
  - CPU, memory, network, disk metrics
  - High-frequency data with proper indexing
- **cost_data**: Billing and cost information
  - Flexible cost allocation with tags
  - Raw data preservation for audit

#### Intelligence & Automation
- **recommendations**: AI-powered optimization suggestions
  - Confidence scoring and priority levels
  - Lifecycle management (pending → applied)
- **actions**: Automated/manual resource actions
  - Full audit trail with parameters and results

#### User Management
- **users**: Secure user authentication
  - Role-based access control
  - Email verification tracking
- **cloud_accounts**: Secure cloud provider connections
  - Encrypted credential storage
  - Sync status monitoring
- **cost_alerts**: Intelligent alerting system
  - Multiple notification channels
  - Flexible condition operators

## Key Design Features

### Performance Optimization
- **Strategic Indexing**: Composite indexes for common query patterns
- **JSONB GIN Indexes**: Fast tag-based filtering
- **Time-series Optimization**: Efficient temporal queries

### Data Integrity
- **Foreign Key Constraints**: Referential integrity
- **CHECK Constraints**: Data validation at database level
- **UUID Primary Keys**: Distributed-friendly identifiers

### Security
- **Encrypted Storage**: Cloud credentials encrypted at rest
- **Input Validation**: Email format and role validation
- **Audit Trail**: Complete action and notification history

### Extensibility
- **JSONB Columns**: Flexible metadata and configuration storage
- **Multi-cloud Support**: Provider-agnostic resource tracking
- **Versioned Schema**: Migration-friendly design

## Migration Usage

### Running Migrations

```bash
# Using golang-migrate (recommended)
migrate -path migrations -database "postgres://..." up

# Manual execution
psql -d devcost_ai -f migrations/002_cloud_resources_schema.up.sql
psql -d devcost_ai -f migrations/003_users_schema.up.sql
```

### Rolling Back

```bash
# Using golang-migrate
migrate -path migrations -database "postgres://..." down

# Manual rollback
psql -d devcost_ai -f migrations/003_users_schema.down.sql
psql -d devcost_ai -f migrations/002_cloud_resources_schema.down.sql
```

## Query Patterns

### Common Queries Optimized

1. **Resource Inventory by Provider**
   ```sql
   SELECT * FROM resources 
   WHERE provider = 'aws' AND region = 'us-east-1'
   ORDER BY created_at DESC;
   ```

2. **Cost Analysis by Tags**
   ```sql
   SELECT SUM(cost_amount) as total_cost
   FROM cost_data
   WHERE tags @> '{"Environment": "production"}'
   AND billing_period_start >= '2026-01-01';
   ```

3. **High-Usage Resources**
   ```sql
   SELECT r.*, AVG(ru.cpu_usage) as avg_cpu
   FROM resources r
   JOIN resource_usage ru ON r.id = ru.resource_id
   WHERE ru.timestamp >= NOW() - INTERVAL '7 days'
   GROUP BY r.id
   HAVING AVG(ru.cpu_usage) > 80;
   ```

4. **Active Recommendations**
   ```sql
   SELECT * FROM recommendations
   WHERE status = 'pending'
   AND expires_at > NOW()
   ORDER BY priority DESC, confidence_score DESC;
   ```

## Monitoring

### Database Health Checks
- Connection pool monitoring
- Query performance tracking
- Index usage statistics
- Table size monitoring

### Recommended Monitoring Queries
```sql
-- Table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) as size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;
```

## Security Considerations

1. **Credential Encryption**: Always encrypt cloud provider credentials
2. **Access Control**: Implement proper user role permissions
3. **Audit Logging**: Track all resource modifications
4. **Data Retention**: Implement appropriate data lifecycle policies

## Performance Tuning

1. **Connection Pooling**: Configure appropriate pool sizes
2. **Query Optimization**: Use EXPLAIN ANALYZE for slow queries
3. **Index Maintenance**: Regular index rebuild and statistics
4. **Partitioning**: Consider time-based partitioning for large tables

## Backup Strategy

1. **Regular Backups**: Daily automated backups
2. **Point-in-Time Recovery**: WAL archiving enabled
3. **Cross-Region Replication**: For disaster recovery
4. **Backup Verification**: Regular restore testing
