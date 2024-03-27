package userhdl

type RegisterShareRequest struct {
	Secret      string `json:"secret"`
	UserEntropy bool   `json:"user_entropy"`
	Salt        string `json:"salt,omitempty"`
	Iterations  int    `json:"iterations,omitempty"`
	Length      int    `json:"length,omitempty"`
	Digest      string `json:"digest,omitempty"`
}

type GetShareResponse struct {
	Secret      string `json:"secret"`
	UserEntropy bool   `json:"user_entropy"`
	Salt        string `json:"salt,omitempty"`
	Iterations  int    `json:"iterations,omitempty"`
	Length      int    `json:"length,omitempty"`
	Digest      string `json:"digest,omitempty"`
}
