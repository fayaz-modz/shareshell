package main

import (
	"fmt"
	"log"
	"regexp"
	"sharec/src/server"
	"sharec/src/terminal"
	"strconv"
)

func main() {
	var bufferLines [][]byte
	currentLine := 0
	currentLetter := 0
	bufferLines = append(bufferLines, make([]byte, 0))

	stopCh := make(chan struct{})
	defer func() {
		// Ensure stopCh is closed only once
		select {
		case <-stopCh:
			// Channel was already closed
		default:
			close(stopCh)
		}
	}()

	// Create a new WebSocket client
	client, errs := server.NewClient("ws://localhost:8080/ws")
	if errs != nil {
		log.Fatal("Error connecting to server:", errs)
	}
	defer client.Close()

	// Goroutine for reading messages from the server
	go func() {
		for msg := range client.Receive() {
			fmt.Println("Received from server:", string(msg))
		}
	}()

	// Send a message to the server
	client.Send([]byte("Hello, Server!"))

	// Goroutine to handle incoming messages
	go func() {
		for msg := range client.Receive() {
            // fmt.Print("\033[2J\033[H")
            print(string(msg))
            // var nbuf  [][]byte
            // for i, b := range msg {
            //     if b == 13 {
            //         currentLine++
            //         currentLetter = 0
            //         nbuf = append(nbuf, make([]byte, 0))
            //     } else {
            //         if i+1 >= len(nbuf) {
            //             nbuf = append(nbuf, make([]byte, 0))
            //         }
            //         nbuf [currentLine] = append(nbuf[currentLine], b)
            //     }
            // }
            // isDiff := false
            // for i := 0; i < len(bufferLines); i++ {
            //     if string(bufferLines[i]) != string(nbuf[i]) {
            //         isDiff = true
            //         break
            //     }
            // }
            // if isDiff {
            //     bufferLines = nbuf
            //     // terminal.RenderBuffer(bufferLines)
            // }
		}
	}()


	err := terminal.TermKeyListener(func(key byte) {
		if key == 13 {
			currentLine++
			currentLetter = 0
			bufferLines = append(bufferLines, make([]byte, 0))
			// Regular expression pattern
			pattern := `^(\d+)([a-zA-Z]+.*)$`

			// Compile the regular expression
			re := regexp.MustCompile(pattern)

			// Find string matches
			matches := re.FindStringSubmatch(string(bufferLines[currentLine-1]))

			if len(matches) > 0 {
				// Extract number and text from capturing groups
				number := matches[1]
				text := matches[2]
				print(text)

				// Output results
				i, _ := strconv.Atoi(number)
				if i+1 <= len(bufferLines) {
					print(i, text)
					bufferLines[i] = []byte(text)
					currentLine--
					bufferLines[currentLine] = []byte("")
				}
			}
		} else {
			currentLetter++
			bufferLines[currentLine] = append(bufferLines[currentLine], key)
		}
		// render(bufferLines)

		if key == '/' {
		          close(stopCh)
			log.Printf("end\n")
		          return
		}

        k := make([]byte, 1)
        k[0] = key
        print(string(k))
        
        client.Send(k)

	}, stopCh)

	if err != nil {
		log.Printf("Error in termKeyListener: %v", err)
	}
	println(err)
    select {}
}
