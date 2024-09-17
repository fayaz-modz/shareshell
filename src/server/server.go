package server

import (
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

type Server struct {
	upgrader  websocket.Upgrader
	isServing bool
	msgChan   chan []byte
	mu        sync.RWMutex
	conn      *websocket.Conn
}

// Singleton instance and lock
var (
	instance *Server
	once     sync.Once
)

// GetInstance returns the singleton instance of the Server
func GetInstance() *Server {
	once.Do(func() {
		instance = &Server{
			upgrader: websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			},
			msgChan: make(chan []byte),
			conn:    nil,
		}
	})
	return instance
}

// Start starts the server and listens for connections
func (s *Server) Start(addr string) error {
	s.mu.Lock()
	s.isServing = true
	s.mu.Unlock()

	http.HandleFunc("/ws", s.handleConnection)
	log.Println("Server started on", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		s.mu.Lock()
		s.isServing = false
		s.mu.Unlock()
		return errors.New("Error starting server: " + err.Error())
	}
	return nil
}

// IsServing returns whether the server is currently serving requests
func (s *Server) IsServing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isServing
}

// Send sends a message to the WebSocket clients
func (s *Server) Send(msg []byte) {
	// print(s)
	// s.conn.ReadMessage()
	if s.conn == nil {
		print("Connection is nil")
	} else {
		s.conn.WriteMessage(websocket.TextMessage, msg)
	}
}

// Receive returns a channel for receiving messages
func (s *Server) Receive() <-chan []byte {
	return s.msgChan
}

// handleConnection handles incoming WebSocket connections
func (s *Server) handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error while upgrading connection:", err)
		return
	}
	defer conn.Close()
	log.Println("Client connected")

	s.conn = conn

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error while reading message:", err)
				return
			}
			s.Send(msg) // Send received message to the msgChan
		}
	}()

	for msg := range s.Receive() {
		print(msg)
		err := conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Error while sending message:", err)
			return
		}
	}
}
