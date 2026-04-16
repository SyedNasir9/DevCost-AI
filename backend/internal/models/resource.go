package models

import (
	"time"

	"github.com/google/uuid"
)

// ResourceType represents the type of cloud resource
type ResourceType string

const (
	ResourceTypeEC2    ResourceType = "EC2"
	ResourceTypeRDS    ResourceType = "RDS"
	ResourceTypeEBS    ResourceType = "EBS"
	ResourceTypeLambda ResourceType = "Lambda"
	ResourceTypeS3     ResourceType = "S3"
	ResourceTypeVPC    ResourceType = "VPC"
	ResourceTypeSubnet ResourceType = "Subnet"
	ResourceTypeEKS    ResourceType = "EKS"
)

// ResourceState represents the state of a resource
type ResourceState string

const (
	ResourceStatePending      ResourceState = "pending"
	ResourceStateRunning      ResourceState = "running"
	ResourceStateStopping     ResourceState = "stopping"
	ResourceStateStopped      ResourceState = "stopped"
	ResourceStateShuttingDown ResourceState = "shutting-down"
	ResourceStateTerminated   ResourceState = "terminated"
	ResourceStateRebooting    ResourceState = "rebooting"
	ResourceStateAvailable    ResourceState = "available"
	ResourceStateCreating     ResourceState = "creating"
	ResourceStateModifying    ResourceState = "modifying"
	ResourceStateDeleting     ResourceState = "deleting"
	ResourceStateBackup       ResourceState = "backup-restoring"
)

// Resource represents a generic cloud resource
type Resource struct {
	ID           string                 `json:"id" db:"id"`
	ResourceID   string                 `json:"resource_id" db:"resource_id"` // Cloud provider specific ID
	ResourceType ResourceType           `json:"resource_type" db:"resource_type"`
	Provider     string                 `json:"provider" db:"provider"` // aws, gcp, azure
	Region       string                 `json:"region" db:"region"`
	AccountID    string                 `json:"account_id" db:"account_id"`
	Name         string                 `json:"name" db:"name"`
	State        ResourceState          `json:"state" db:"state"`
	InstanceType string                 `json:"instance_type,omitempty" db:"instance_type"`
	Tags         map[string]string      `json:"tags" db:"tags"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// NewResource creates a new resource with default values
func NewResource(resourceID, resourceType, provider, region, accountID string) *Resource {
	now := time.Now()
	return &Resource{
		ID:           uuid.New().String(),
		ResourceID:   resourceID,
		ResourceType: ResourceType(resourceType),
		Provider:     provider,
		Region:       region,
		AccountID:    accountID,
		Tags:         make(map[string]string),
		Metadata:     make(map[string]interface{}),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// EC2Resource represents an EC2-specific resource with additional fields
type EC2Resource struct {
	Resource
	InstanceType       string    `json:"instance_type"`
	AvailabilityZone   string    `json:"availability_zone"`
	SubnetID           string    `json:"subnet_id"`
	VpcID              string    `json:"vpc_id"`
	SecurityGroups     []string  `json:"security_groups"`
	KeyName            string    `json:"key_name"`
	LaunchTime         time.Time `json:"launch_time"`
	PublicIP           string    `json:"public_ip"`
	PrivateIP          string    `json:"private_ip"`
	Platform           string    `json:"platform"`
	Architecture       string    `json:"architecture"`
	Hypervisor         string    `json:"hypervisor"`
	VirtualizationType string    `json:"virtualization_type"`
	Lifecycle          string    `json:"lifecycle"`
	MonitoringState    string    `json:"monitoring_state"`
}

// RDSResource represents an RDS-specific resource with additional fields
type RDSResource struct {
	Resource
	Engine                     string   `json:"engine"`
	EngineVersion              string   `json:"engine_version"`
	InstanceClass              string   `json:"instance_class"`
	Status                     string   `json:"status"`
	Endpoint                   Endpoint `json:"endpoint"`
	AvailabilityZone           string   `json:"availability_zone"`
	SubnetGroupName            string   `json:"subnet_group_name"`
	SubnetGroupVpcID           string   `json:"subnet_group_vpc_id"`
	VpcSecurityGroups          []string `json:"vpc_security_groups"`
	ParameterGroupName         string   `json:"parameter_group_name"`
	OptionGroupName            string   `json:"option_group_name"`
	MultiAZ                    bool     `json:"multi_az"`
	PubliclyAccessible         bool     `json:"publicly_accessible"`
	StorageType                string   `json:"storage_type"`
	AllocatedStorage           int32    `json:"allocated_storage"`
	StorageEncrypted           bool     `json:"storage_encrypted"`
	BackupRetentionPeriod      int32    `json:"backup_retention_period"`
	PreferredBackupWindow      string   `json:"preferred_backup_window"`
	PreferredMaintenanceWindow string   `json:"preferred_maintenance_window"`
	AutoMinorVersionUpgrade    bool     `json:"auto_minor_version_upgrade"`
	LicenseModel               string   `json:"license_model"`
}

// Endpoint represents RDS endpoint information
type Endpoint struct {
	Address      string `json:"address"`
	Port         int32  `json:"port"`
	HostedZoneId string `json:"hosted_zone_id"`
}

// NewRDSResource creates a new RDS resource
func NewRDSResource(dbInstanceID, instanceClass, region, accountID string) *RDSResource {
	resource := NewResource(dbInstanceID, string(ResourceTypeRDS), "aws", region, accountID)
	rdsResource := &RDSResource{
		Resource:          *resource,
		InstanceClass:     instanceClass,
		VpcSecurityGroups: []string{},
	}
	return rdsResource
}

// ToResource converts RDSResource to base Resource
func (r *RDSResource) ToResource() *Resource {
	// Copy base resource fields
	resource := &Resource{
		ID:           r.ID,
		ResourceID:   r.ResourceID,
		ResourceType: r.ResourceType,
		Provider:     r.Provider,
		Region:       r.Region,
		AccountID:    r.AccountID,
		Name:         r.Name,
		State:        r.State,
		InstanceType: r.InstanceClass,
		Tags:         r.Tags,
		Metadata:     make(map[string]interface{}),
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}

	// Add RDS-specific metadata
	resource.Metadata["engine"] = r.Engine
	resource.Metadata["engine_version"] = r.EngineVersion
	resource.Metadata["endpoint"] = r.Endpoint
	resource.Metadata["availability_zone"] = r.AvailabilityZone
	resource.Metadata["subnet_group_name"] = r.SubnetGroupName
	resource.Metadata["subnet_group_vpc_id"] = r.SubnetGroupVpcID
	resource.Metadata["vpc_security_groups"] = r.VpcSecurityGroups
	resource.Metadata["parameter_group_name"] = r.ParameterGroupName
	resource.Metadata["option_group_name"] = r.OptionGroupName
	resource.Metadata["multi_az"] = r.MultiAZ
	resource.Metadata["publicly_accessible"] = r.PubliclyAccessible
	resource.Metadata["storage_type"] = r.StorageType
	resource.Metadata["allocated_storage"] = r.AllocatedStorage
	resource.Metadata["storage_encrypted"] = r.StorageEncrypted
	resource.Metadata["backup_retention_period"] = r.BackupRetentionPeriod
	resource.Metadata["preferred_backup_window"] = r.PreferredBackupWindow
	resource.Metadata["preferred_maintenance_window"] = r.PreferredMaintenanceWindow
	resource.Metadata["auto_minor_version_upgrade"] = r.AutoMinorVersionUpgrade
	resource.Metadata["license_model"] = r.LicenseModel

	return resource
}

// EBSResource represents an EBS-specific resource with additional fields
type EBSResource struct {
	Resource
	Size             int32      `json:"size"`
	VolumeType       string     `json:"volume_type"`
	Iops             int32      `json:"iops"`
	Throughput       int32      `json:"throughput"`
	Encrypted        bool       `json:"encrypted"`
	AvailabilityZone string     `json:"availability_zone"`
	SnapshotID       string     `json:"snapshot_id"`
	IsAttached       bool       `json:"is_attached"`
	Attachment       Attachment `json:"attachment"`
	CreateTime       time.Time  `json:"create_time"`
}

// Attachment represents EBS volume attachment information
type Attachment struct {
	InstanceID string    `json:"instance_id"`
	Device     string    `json:"device"`
	State      string    `json:"state"`
	AttachTime time.Time `json:"attach_time"`
}

// NewEBSResource creates a new EBS resource
func NewEBSResource(volumeID string, size int32, region, accountID string) *EBSResource {
	resource := NewResource(volumeID, string(ResourceTypeEBS), "aws", region, accountID)
	ebsResource := &EBSResource{
		Resource: *resource,
		Size:     size,
	}
	return ebsResource
}

// ToResource converts EBSResource to base Resource
func (e *EBSResource) ToResource() *Resource {
	// Copy base resource fields
	resource := &Resource{
		ID:           e.ID,
		ResourceID:   e.ResourceID,
		ResourceType: e.ResourceType,
		Provider:     e.Provider,
		Region:       e.Region,
		AccountID:    e.AccountID,
		Name:         e.Name,
		State:        e.State,
		Tags:         e.Tags,
		Metadata:     make(map[string]interface{}),
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}

	// Add EBS-specific metadata
	resource.Metadata["size"] = e.Size
	resource.Metadata["volume_type"] = e.VolumeType
	resource.Metadata["iops"] = e.Iops
	resource.Metadata["throughput"] = e.Throughput
	resource.Metadata["encrypted"] = e.Encrypted
	resource.Metadata["availability_zone"] = e.AvailabilityZone
	resource.Metadata["snapshot_id"] = e.SnapshotID
	resource.Metadata["is_attached"] = e.IsAttached
	resource.Metadata["attachment"] = e.Attachment
	resource.Metadata["create_time"] = e.CreateTime

	return resource
}

// ToResource converts EC2Resource to base Resource
func (e *EC2Resource) ToResource() *Resource {
	// Copy base resource fields
	resource := &Resource{
		ID:           e.ID,
		ResourceID:   e.ResourceID,
		ResourceType: e.ResourceType,
		Provider:     e.Provider,
		Region:       e.Region,
		AccountID:    e.AccountID,
		Name:         e.Name,
		State:        e.State,
		InstanceType: e.InstanceType,
		Tags:         e.Tags,
		Metadata:     make(map[string]interface{}),
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}

	// Add EC2-specific metadata
	resource.Metadata["availability_zone"] = e.AvailabilityZone
	resource.Metadata["subnet_id"] = e.SubnetID
	resource.Metadata["vpc_id"] = e.VpcID
	resource.Metadata["security_groups"] = e.SecurityGroups
	resource.Metadata["key_name"] = e.KeyName
	resource.Metadata["launch_time"] = e.LaunchTime
	resource.Metadata["public_ip"] = e.PublicIP
	resource.Metadata["private_ip"] = e.PrivateIP
	resource.Metadata["platform"] = e.Platform
	resource.Metadata["architecture"] = e.Architecture
	resource.Metadata["hypervisor"] = e.Hypervisor
	resource.Metadata["virtualization_type"] = e.VirtualizationType
	resource.Metadata["lifecycle"] = e.Lifecycle
	resource.Metadata["monitoring_state"] = e.MonitoringState

	return resource
}

// ResourceFilter represents filters for resource queries
type ResourceFilter struct {
	ResourceTypes []ResourceType    `json:"resource_types"`
	States        []ResourceState   `json:"states"`
	Regions       []string          `json:"regions"`
	AccountIDs    []string          `json:"account_ids"`
	Tags          map[string]string `json:"tags"`
	Limit         int               `json:"limit"`
	Offset        int               `json:"offset"`
}

// ResourceList represents a paginated list of resources
type ResourceList struct {
	Resources []*Resource `json:"resources"`
	Total     int         `json:"total"`
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
	HasNext   bool        `json:"has_next"`
}

// ResourceMetrics represents metrics for a resource
type ResourceMetrics struct {
	ResourceID string                 `json:"resource_id"`
	MetricType string                 `json:"metric_type"`
	Timestamp  time.Time              `json:"timestamp"`
	Value      float64                `json:"value"`
	Unit       string                 `json:"unit"`
	Labels     map[string]string      `json:"labels"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ResourceCost represents cost information for a resource
type ResourceCost struct {
	ResourceID    string            `json:"resource_id"`
	CostAmount    float64           `json:"cost_amount"`
	Currency      string            `json:"currency"`
	BillingPeriod time.Time         `json:"billing_period"`
	ServiceName   string            `json:"service_name"`
	UsageType     string            `json:"usage_type"`
	UsageQuantity float64           `json:"usage_quantity"`
	Rate          float64           `json:"rate"`
	Tags          map[string]string `json:"tags"`
}

// ResourceRecommendation represents a cost optimization recommendation
type ResourceRecommendation struct {
	ID               string                 `json:"id"`
	ResourceID       string                 `json:"resource_id"`
	Type             string                 `json:"type"`
	Priority         string                 `json:"priority"`
	Title            string                 `json:"title"`
	Description      string                 `json:"description"`
	EstimatedSavings float64                `json:"estimated_savings"`
	ConfidenceScore  float64                `json:"confidence_score"`
	CurrentState     map[string]interface{} `json:"current_state"`
	RecommendedState map[string]interface{} `json:"recommended_state"`
	Status           string                 `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// Helper functions

// IsActive checks if a resource is in an active state
func (r *Resource) IsActive() bool {
	switch r.ResourceType {
	case ResourceTypeEC2:
		return r.State == ResourceStateRunning
	case ResourceTypeRDS:
		return r.State == ResourceStateAvailable
	default:
		return r.State != ResourceStateTerminated && r.State != ResourceStateDeleting
	}
}

// GetTag returns a tag value by key
func (r *Resource) GetTag(key string) (string, bool) {
	value, exists := r.Tags[key]
	return value, exists
}

// SetTag sets a tag value
func (r *Resource) SetTag(key, value string) {
	if r.Tags == nil {
		r.Tags = make(map[string]string)
	}
	r.Tags[key] = value
	r.UpdatedAt = time.Now()
}

// RemoveTag removes a tag
func (r *Resource) RemoveTag(key string) {
	delete(r.Tags, key)
	r.UpdatedAt = time.Now()
}

// HasTag checks if a resource has a specific tag
func (r *Resource) HasTag(key string) bool {
	_, exists := r.Tags[key]
	return exists
}

// GetMetadata returns metadata value by key
func (r *Resource) GetMetadata(key string) (interface{}, bool) {
	if r.Metadata == nil {
		return nil, false
	}
	value, exists := r.Metadata[key]
	return value, exists
}

// SetMetadata sets metadata value
func (r *Resource) SetMetadata(key string, value interface{}) {
	if r.Metadata == nil {
		r.Metadata = make(map[string]interface{})
	}
	r.Metadata[key] = value
	r.UpdatedAt = time.Now()
}
