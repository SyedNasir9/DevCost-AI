package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email        string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"not null" json:"-"`
	FirstName    string     `gorm:"size:100" json:"first_name"`
	LastName     string     `gorm:"size:100" json:"last_name"`
	Role         string     `gorm:"default:user" json:"role"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the table name for the User model
func (User) TableName() string {
	return "users"
}

// CloudAccount represents a cloud provider account
type CloudAccount struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID            uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Provider          string     `gorm:"not null" json:"provider"` // aws, gcp, azure
	AccountName       string     `gorm:"not null" json:"account_name"`
	AccountID         string     `gorm:"not null" json:"account_id"`
	CredentialsEncrypted string   `gorm:"type:text" json:"-"`
	IsActive          bool       `gorm:"default:true" json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relationships
	User              User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CostData          []CostData `gorm:"foreignKey:CloudAccountID" json:"cost_data,omitempty"`
	Recommendations   []Recommendation `gorm:"foreignKey:CloudAccountID" json:"recommendations,omitempty"`
}

// TableName returns the table name for the CloudAccount model
func (CloudAccount) TableName() string {
	return "cloud_accounts"
}
