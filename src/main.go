package main

import (
	"fmt"
	"log"
	"sharec/src/server"
	"sharec/src/terminal"
	"sync"

	"time"
)

func main() {
	// var bufferLines [][]byte
	// bufferLines = append(bufferLines, make([]byte, 0))

	handler := terminal.NewTerminalHandler()

	stopChannel := make(chan bool)

	// Get the singleton server instance
	srv := server.GetInstance()
	//
	// Create a wait group to wait for the server and message processing
	var wg sync.WaitGroup

	// Start the server in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done() // Mark as done when the goroutine finishes

		if err := srv.Start(":8080"); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Handle receiving messages from clients
	wg.Add(1)
	go func() {
		defer wg.Done() // Mark as done when the goroutine finishes

		for msg := range srv.Receive() {
	           fmt.Println(string(msg))
	           // handler.InsertInput(msg)
		//           c := 0
		//           for _, b := range msg {
		//               if b == 13 {
		//                   c++;
		//               } else {
		//                   if c >= len(bufferLines) {
		//                       bufferLines = append(bufferLines, make([]byte, 0))
		//                   }
		//                   bufferLines[c] = append(bufferLines[c], b)
		//               }
		//           }
		//           terminal.RenderBuffer(bufferLines)
		//           if handler.IsReady() {
		//               print(string(msg))
		//               print("\r")
		//               handler.InsertInput(msg)
		//           }
		}
	}()

	go func() {
		for {
			select {
			case <-stopChannel:
				return
			default:
				if handler.IsReady() {
					close(stopChannel)
					handler.InsertInput([]byte("ls"))
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	   go func() {
	       for range handler.StreamBufferChanges() {
	           srv.Send(handler.GetScreenBuffer())
	       }
	   }()

	if err := terminal.RunTerminal(handler); err != nil {
		log.Fatalf("Error running terminal: %v", err)
	}
	// Wait for all goroutines to complete
	// wg.Wait()
}

// waitForServerReady blocks until the server is ready
func waitForServerReady(srv *server.Server) {
	for !srv.IsServing() {
		// Use a select with a timeout to avoid busy-waiting
		select {
		case <-time.After(100 * time.Millisecond):
			// Check if the server is serving
			if srv.IsServing() {
				log.Println("Server is now ready")
				return
			}
		}
	}
}
