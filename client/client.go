package client

import (
	"context"
	"fmt"
	"log"
	"net"

	"y3cache/proto"
)

type (
	Options struct{}
	Client  struct {
		conn net.Conn
	}
)

func NewFromConn(conn net.Conn) *Client {
	return &Client{
		conn: conn,
	}
}

func (c *Client) Get(ctx context.Context, key []byte) ([]byte, error) {
	cmd := &proto.CommandGet{
		Key: key,
	}
	_, err := c.conn.Write(cmd.Bytes())
	if err != nil {
		return nil, err
	}
	resp, err := proto.ParseGetResponse(c.conn)
	if err != nil {
		return nil, err
	}
	if resp.Status == proto.StatusKeyNotFound {
		return nil, fmt.Errorf("could not find key (%s)", key)
	}

	if resp.Status != proto.StatusOK {
		return nil, fmt.Errorf(
			"server repsonsed with non OK status [%s]",
			resp.Status,
		)
	}
	return resp.Value, nil
}

func (c *Client) Set(ctx context.Context, key, value []byte) error {
	cmd := &proto.CommandSet{
		Key:   key,
		Value: value,
		TTL:   0,
	}
	_, err := c.conn.Write(cmd.Bytes())
	if err != nil {
		return err
	}

	resp, err := proto.ParseSetResponse(c.conn)
	fmt.Println("received: ", string(resp.Bytes()))
	if err != nil {
		return err
	}
	if resp.Status != proto.StatusOK {
		return fmt.Errorf(
			"server repsonsed with non OK status [%s]",
			resp.Status,
		)
	}
	return nil
}

func New(endpoint string, _ Options) (*Client, error) {
	conn, err := net.Dial("tcp", endpoint)
	if err != nil {
		log.Fatal(err)
	}

	return &Client{
		conn: conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
