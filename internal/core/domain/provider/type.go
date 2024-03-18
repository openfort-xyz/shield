package provider

type Type uint8

const (
	TypeUnknown Type = iota
	TypeOpenfort
	TypeSupabase
	TypeCustom
)

func (t Type) String() string {
	switch t {
	case TypeOpenfort:
		return "OPENFORT"
	case TypeSupabase:
		return "SUPABASE"
	case TypeCustom:
		return "CUSTOM"
	default:
		return "UNKNOWN"
	}
}
