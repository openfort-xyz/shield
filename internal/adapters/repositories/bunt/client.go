package bunt

import "github.com/tidwall/buntdb"

type Client struct {
	*buntdb.DB
}

var singleton *buntdb.DB

func New() (*Client, error) {
	if singleton != nil {
		return &Client{
			DB: singleton,
		}, nil
	}

	db, err := buntdb.Open(":memory:")
	if err != nil {
		return nil, err
	}

	singleton = db

	return &Client{
		DB: db,
	}, nil
}

func (c *Client) Close() error {
	return c.DB.Close()
}
