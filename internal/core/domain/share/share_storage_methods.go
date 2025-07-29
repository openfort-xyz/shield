package share

type StorageMethodID int32

const (
	StorageMethodShield StorageMethodID = iota
	StorageMethodGoogleDrive
	StorageMethodICloud
)

type StorageMethod struct {
	ID   int32
	Name string
}
