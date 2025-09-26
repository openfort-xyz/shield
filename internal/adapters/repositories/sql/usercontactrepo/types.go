package usercontactrepo

type UserContact struct {
	ID             int    `gorm:"column:id;primaryKey"`
	ExternalUserID string `gorm:"external_user_id"`
	Email          string `gorm:"email"`
	Phone          string `gorm:"phone"`
}

func (UserContact) TableName() string {
	return "shld_user_contacts"
}
