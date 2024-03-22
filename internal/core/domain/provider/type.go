package provider

type Type uint8

const (
	TypeUnknown Type = iota
	TypeOpenfort
	TypeCustom
)

func (t Type) String() string {
	switch t {
	case TypeOpenfort:
		return "OPENFORT"
	case TypeCustom:
		return "CUSTOM"
	default:
		return "UNKNOWN"
	}
}
