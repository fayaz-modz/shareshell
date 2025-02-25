package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/google/btree"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	messages   = btree.New(4)
	messagesMu sync.Mutex
)

func main() {
	http.HandleFunc("/offer", handleOffer)
	http.HandleFunc("/connect", handleConnect)
	
  port := os.Getenv("PORT")
	if port == "" {
    port = "8080"
	}

	fmt.Println("Server started on port", port)
  if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Println("Could not start server: ", err)
	}
}
