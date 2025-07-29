package share

type ShareStorageMethodID int32

const (
	StorageMethodShield ShareStorageMethodID = iota + 1
	StorageMethodGoogleDrive
	StorageMethodICloud
)

type ShareStorageMethod struct {
	ID   int32
	Name string
}
