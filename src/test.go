package main

import (
    "log"
    "net/http"
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Error while upgrading connection:", err)
        return
    }
    defer conn.Close()
    log.Println("Client connected")

    for {
        msgType, msg, err := conn.ReadMessage()
        if err != nil {
            log.Println("Error while reading message:", err)
            break
        }
        log.Printf("Received message: %s", msg)

        err = conn.WriteMessage(msgType, msg)
        if err != nil {
            log.Println("Error while sending message:", err)
            break
        }
    }
}

func main() {
    http.HandleFunc("/ws", handleConnection)
    serverAddr := "localhost:8080"
    log.Printf("WebSocket server started at ws://%s", serverAddr)
    if err := http.ListenAndServe(serverAddr, nil); err != nil {
        log.Fatal("Error starting server:", err)
    }
}

