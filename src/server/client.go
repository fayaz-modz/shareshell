package server

import (
	"github.com/gorilla/websocket"
	"log"
)

type Client struct {
	conn   *websocket.Conn
	sendCh chan []byte
	recvCh chan []byte
}

// NewClient connects to the WebSocket server and creates a new Client
func NewClient(url string) (*Client, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	client := &Client{
		conn:   conn,
		sendCh: make(chan []byte),
		recvCh: make(chan []byte),
	}

	// Start reading and writing messages in separate goroutines
	go client.readMessages()
	go client.writeMessages()

	return client, nil
}

// Send pushes a message into the send channel for sending to the server
func (c *Client) Send(message []byte) {
	c.sendCh <- message
}

// Receive returns a channel for receiving messages from the server
func (c *Client) Receive() <-chan []byte {
	return c.recvCh
}

// Close closes the WebSocket connection and stops all goroutines
func (c *Client) Close() {
	c.conn.Close()
	close(c.sendCh)
	close(c.recvCh)
}

// readMessages continuously reads messages from the WebSocket server
func (c *Client) readMessages() {
	defer close(c.recvCh)
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message:", err)
			return
		}
		c.recvCh <- msg // Push received message to the recv channel
	}
}

// writeMessages continuously sends messages from the send channel to the server
func (c *Client) writeMessages() {
	for msg := range c.sendCh {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Error while sending message:", err)
			return
		}
	}
}
