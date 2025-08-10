package p2p

import (
	"context"
	"net"
)

type Client struct {
	encoder Encoder

	peers map[string]Peer
}

func (c *Client) Send(ctx context.Context, msg Message, address string) error {

	peer, ok := c.peers[address]

	if !ok {

		conn, err := c.connectTo(address)

		if err != nil {
			return err
		}

		peer, err := NewTCPPeer(conn)

		if err != nil {
			return err
		}

		c.peers[address] = peer

	}

	return c.sendMessage(ctx, peer, msg)
}

func (c *Client) connectTo(address string) (net.Conn, error) {
	conn, err := net.Dial("tcp", address)

	if err != nil {
		return nil, err
	}

	return conn, err

}

func (c *Client) sendMessage(ctx context.Context, peer Peer, msg Message) error {
	byteMsg, err := c.encoder.Encode(msg)

	if err != nil {
		return err
	}

	err = peer.Send(ctx, byteMsg)

	if err != nil {
		return err
	}

	return nil
}
