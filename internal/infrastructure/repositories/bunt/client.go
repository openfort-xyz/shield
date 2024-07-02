package bunt

import "github.com/tidwall/buntdb"

type Client struct {
	*buntdb.DB
}

func New() (*Client, error) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		return nil, err
	}

	return &Client{
		DB: db,
	}, nil
}

func (c *Client) Close() error {
	return c.DB.Close()
}
