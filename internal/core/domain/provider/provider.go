package provider

type Provider struct {
	ID        string
	ProjectID string
	Type      Type
	Config    interface{}
}
