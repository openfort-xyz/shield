package providerrepo

import (
	"gorm.io/gorm"
	"time"
)

type Provider struct {
	ID        string         `gorm:"column:id;primary_key"`
	ProjectID string         `gorm:"column:project_id;not null"`
	Type      Type           `gorm:"column:type;not null"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (Provider) TableName() string {
	return "shld_providers"
}

type Type string

const (
	TypeOpenfort Type = "OPENFORT"
	TypeSupabase Type = "SUPABASE"
	TypeCustom   Type = "CUSTOM"
)

type ProviderOpenfort struct {
	ProviderID      string `gorm:"column:provider_id;primary_key"`
	OpenfortProject string `gorm:"column:openfort_project"`
}

func (ProviderOpenfort) TableName() string {
	return "shld_openfort_providers"
}

type ProviderSupabase struct {
	ProviderID      string `gorm:"column:provider_id;primary_key"`
	SupabaseProject string `gorm:"column:supabase_project"`
}

func (ProviderSupabase) TableName() string {
	return "shld_supabase_providers"
}

type ProviderCustom struct {
	ProviderID string `gorm:"column:provider_id;primary_key"`
	JWKUrl     string `gorm:"column:jwk_url"`
}

func (ProviderCustom) TableName() string {
	return "shld_custom_providers"
}
