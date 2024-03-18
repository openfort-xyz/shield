package provider

type Type uint8

const (
	TypeUnknown Type = iota
	TypeOpenfort
	TypeSupabase
	TypeCustom
)
