package sharerepo

import (
	"time"

	"gorm.io/gorm"
)

type Share struct {
	ID                   string               `gorm:"column:id;primary_key"`
	Data                 string               `gorm:"column:data; not null"`
	UserID               *string              `gorm:"column:user_id;default:null"`
	Entropy              Entropy              `gorm:"column:entropy;default:none"`
	Salt                 string               `gorm:"column:salt;default:null"`
	Iterations           int                  `gorm:"column:iterations;default:null"`
	Length               int                  `gorm:"column:length;default:null"`
	Digest               string               `gorm:"column:digest;default:null"`
	KeyChainID           *string              `gorm:"column:keychain_id;default:null"`
	Reference            *string              `gorm:"column:reference;default:null"`
	ShareStorageMethodID ShareStorageMethodID `gorm:"column:storage_method_id;not null"`
	ShareStorageMethod   *ShareStorageMethod  `gorm:"foreignKey:ShareStorageMethodID"`
	PasskeyReference     *PasskeyReference    `gorm:"foreignKey:ShareReference;references:ID"`
	CreatedAt            time.Time            `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time            `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt            gorm.DeletedAt       `gorm:"column:deleted_at"`
}

func (Share) TableName() string {
	return "shld_shares"
}

type Entropy string

const (
	EntropyNone    Entropy = "none"
	EntropyUser    Entropy = "user"
	EntropyProject Entropy = "project"
	EntropyPasskey Entropy = "passkey"
)

type ShareStorageMethodID int32

const (
	StorageMethodShield ShareStorageMethodID = iota
	StorageMethodGoogleDrive
	StorageMethodICloud
)

type ShareStorageMethod struct {
	ID   int32  `gorm:"column:id;primary_key"`
	Name string `gorm:"column:name;not null"`
}

func (ShareStorageMethod) TableName() string {
	return "shld_share_storage_methods"
}

type PasskeyReference struct {
	PasskeyID      string `gorm:"column:passkey_id; primary_key"`
	PasskeyEnv     string `gorm:"column:passkey_env; not null"`
	ShareReference string `gorm:"column:share_reference; not null"`
	Share          Share  `gorm:"foreignKey:ShareReference;references:ID"`
}

func (PasskeyReference) TableName() string {
	return "shld_passkey_references"
}

type InfoByReference struct {
	Reference  string
	Entropy    Entropy
	PasskeyID  *string
	PasskeyEnv *string
}

type InfoByUserID struct {
	UserID     string
	Entropy    Entropy
	PasskeyID  *string
	PasskeyEnv *string
}
