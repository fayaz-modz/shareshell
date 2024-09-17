package terminal

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/term"
)

func RenderBuffer(buffer [][]byte) {
    // Clear the screen and move cursor to top-left corner
    fmt.Print("\033[2J\033[H")
    
    // Print each line of the buffer
    for i, line := range buffer {
        if i > 0 {
            // Use both carriage return and line feed to start a new line
            fmt.Print("\r\n")
        }
        // Print the line without any additional formatting
        fmt.Print(string(line))
    }
    
    // Ensure the cursor is at the start of the next line
    fmt.Print("\r")
}



// TermKeyListener listens for key events and processes them, stopping when a signal is received.
func TermKeyListener(handleKey func(byte), stopCh <-chan struct{}) error {
	// Set terminal to raw mode
	oldState, err := term.MakeRaw(int(syscall.Stdin))
	if err != nil {
		return errors.New("terminal cannot be set to raw mode")
	}
	defer term.Restore(int(syscall.Stdin), oldState) // Restore terminal when done

	// 1 byte buffer for reading keyboard input
	var buf = make([]byte, 1)

	for {
		select {
		case <-stopCh:
			// Exit when cancellation is signaled
			return nil
		default:
			// Listen for keyboard input events
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return errors.New("error reading from stdin")
			}

			// Handle each key press event
			handleKey(buf[0])
		}
	}
}
