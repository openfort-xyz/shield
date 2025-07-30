package provider

type CustomConfig struct {
	ProviderID      string
	JWK             string
	PEM             string
	CookieFieldName *string
	KeyType         KeyType
}

type KeyType int8

const (
	KeyTypeUnknown KeyType = iota
	KeyTypeRSA
	KeyTypeECDSA
	KeyTypeEd25519
)
