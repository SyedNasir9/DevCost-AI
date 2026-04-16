package models

import (
	"time"

	"github.com/google/uuid"
)

// CostData represents cost information from cloud providers
type CostData struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CloudAccountID     uuid.UUID `gorm:"type:uuid;not null" json:"cloud_account_id"`
	ServiceName        string    `gorm:"not null" json:"service_name"`
	ResourceType       string    `json:"resource_type"`
	CostAmount         float64   `gorm:"type:decimal(12,4);not null" json:"cost_amount"`
	Currency           string    `gorm:"default:USD" json:"currency"`
	UsageQuantity      float64   `gorm:"type:decimal(12,4)" json:"usage_quantity"`
	UsageUnit          string    `gorm:"size:100" json:"usage_unit"`
	BillingPeriodStart time.Time `gorm:"not null" json:"billing_period_start"`
	BillingPeriodEnd   time.Time `gorm:"not null" json:"billing_period_end"`
	Tags               string    `gorm:"type:jsonb" json:"tags"`
	RawData            string    `gorm:"type:jsonb" json:"raw_data"`
	CreatedAt          time.Time `json:"created_at"`

	// Relationships
	CloudAccount CloudAccount `gorm:"foreignKey:CloudAccountID" json:"cloud_account,omitempty"`
}

// TableName returns the table name for the CostData model
func (CostData) TableName() string {
	return "cost_data"
}

// Recommendation represents cost optimization recommendations
type Recommendation struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CloudAccountID     uuid.UUID  `gorm:"type:uuid;not null" json:"cloud_account_id"`
	RecommendationType string     `gorm:"not null" json:"recommendation_type"`
	ResourceID         string     `gorm:"not null" json:"resource_id"`
	ResourceName       string     `json:"resource_name"`
	CurrentCost        float64    `gorm:"type:decimal(12,4)" json:"current_cost"`
	PotentialSavings   float64    `gorm:"type:decimal(12,4)" json:"potential_savings"`
	ConfidenceScore    float64    `gorm:"type:decimal(3,2)" json:"confidence_score"`
	Status             string     `gorm:"default:pending" json:"status"`
	Details            string     `gorm:"type:jsonb" json:"details"`
	CreatedAt          time.Time  `json:"created_at"`
	ImplementedAt      *time.Time `json:"implemented_at,omitempty"`

	// Relationships
	CloudAccount CloudAccount `gorm:"foreignKey:CloudAccountID" json:"cloud_account,omitempty"`
}

// TableName returns the table name for the Recommendation model
func (Recommendation) TableName() string {
	return "recommendations"
}

// CostAlert represents cost alerts and notifications
type CostAlert struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	CloudAccountID  *uuid.UUID `gorm:"type:uuid" json:"cloud_account_id,omitempty"`
	AlertType       string     `gorm:"not null" json:"alert_type"`
	ThresholdAmount float64    `gorm:"type:decimal(12,4)" json:"threshold_amount"`
	CurrentAmount   float64    `gorm:"type:decimal(12,4)" json:"current_amount"`
	Message         string     `gorm:"not null" json:"message"`
	IsRead          bool       `gorm:"default:false" json:"is_read"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`

	// Relationships
	User         User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CloudAccount *CloudAccount `gorm:"foreignKey:CloudAccountID" json:"cloud_account,omitempty"`
}

// TableName returns the table name for the CostAlert model
func (CostAlert) TableName() string {
	return "cost_alerts"
}
