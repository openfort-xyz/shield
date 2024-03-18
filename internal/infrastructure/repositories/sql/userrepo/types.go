package userrepo

type User struct {
	ID        string `gorm:"column:id;primaryKey"`
	ProjectID string `gorm:"column:project_id;not null"`
}

func (User) TableName() string {
	return "shld_users"
}

type ExternalUser struct {
	ID             string `gorm:"column:id;primaryKey"`
	UserID         string `gorm:"column:user_id;not null"`
	ExternalUserID string `gorm:"column:external_user_id;not null"`
	Type           Type   `gorm:"column:type;not null"`
	ProjectID      string `gorm:"column:project_id;not null"`
}

func (ExternalUser) TableName() string {
	return "shld_external_users"
}

type Type string

const (
	TypeOpenfort Type = "OPENFORT"
	TypeSupabase Type = "SUPABASE"
	TypeCustom   Type = "CUSTOM"
)
