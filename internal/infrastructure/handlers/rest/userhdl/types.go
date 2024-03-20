package userhdl

type RegisterShareRequest struct {
	Share string `json:"share"`
}

type GetShareResponse struct {
	Share string `json:"share"`
}
