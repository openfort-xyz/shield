package share

type ShareStorageMethodID int32

const (
	StorageMethodShield ShareStorageMethodID = iota
	StorageMethodGoogleDrive
	StorageMethodICloud
)

type ShareStorageMethod struct {
	ID   int32
	Name string
}
