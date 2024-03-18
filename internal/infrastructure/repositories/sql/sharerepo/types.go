package sharerepo

type Share struct {
	ID     string `gorm:"column:id;primary_key"`
	Data   string `gorm:"column:data; not null"`
	UserID string `gorm:"column:user_id;not null"`
}

func (s *Share) TableName() string {
	return "shld_shares"
}
