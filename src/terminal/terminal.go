package terminal

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type TerminalHandler struct {
	ptmx          *os.File
	isReady       bool
	mu            sync.Mutex
	screenBuffer  bytes.Buffer    // Buffer to store current screen characters
	bufferChanged chan struct{}   // Channel to signal buffer changes
}

func NewTerminalHandler() *TerminalHandler {
	return &TerminalHandler{
		bufferChanged: make(chan struct{}, 1), // Initialize the buffer change channel
	}
}

func (th *TerminalHandler) SetReady() {
	th.mu.Lock()
	defer th.mu.Unlock()
	th.isReady = true
}

func (th *TerminalHandler) IsReady() bool {
	th.mu.Lock()
	defer th.mu.Unlock()
	return th.isReady
}

func (th *TerminalHandler) InsertInput(data []byte) {
	if th.ptmx != nil && th.IsReady() {
		_, _ = th.ptmx.Write(data)
	}
}

func (th *TerminalHandler) GetOutput(buf []byte) (int, error) {
	if th.ptmx == nil {
		return 0, fmt.Errorf("pty is not initialized")
	}
	return th.ptmx.Read(buf)
}

func (th *TerminalHandler) GetScreenBuffer() []byte {
	th.mu.Lock()
	defer th.mu.Unlock()
	return th.screenBuffer.Bytes()
}

// StreamBufferChanges allows clients to listen for buffer changes
func (th *TerminalHandler) StreamBufferChanges() <-chan struct{} {
	return th.bufferChanged
}

// notifyBufferChange signals that the buffer has changed
func (th *TerminalHandler) notifyBufferChange() {
	// Non-blocking send to channel to avoid blocking if no one is listening
	select {
	case th.bufferChanged <- struct{}{}:
	default:
	}
}

// RunTerminal function definition
func RunTerminal(th *TerminalHandler) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}

	c := exec.Command(shell)
	ptmx, err := pty.Start(c)
	if err != nil {
		return err
	}
	defer ptmx.Close()

	th.ptmx = ptmx

	// Handle terminal size changes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize
	defer func() { signal.Stop(ch); close(ch) }()

	// Set stdin in raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	// Notify that the PTY is ready
	th.SetReady()

	// Capture pty output in screen buffer and forward it to stdout
	go func() {
		multiWriter := io.MultiWriter(os.Stdout, &th.screenBuffer)
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				// Write to stdout and buffer
				_, _ = multiWriter.Write(buf[:n])
				// Notify about buffer change
				th.notifyBufferChange()
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Printf("error reading from pty: %s", err)
			}
		}
	}()

	// Create a stop channel to signal termination
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

	// Use termKeyListener to read input and send to pty
	go func() {
		err := TermKeyListener(func(key byte) {
			_, writeErr := th.ptmx.Write([]byte{key})
			if writeErr != nil {
				log.Printf("Error writing to pty: %v", writeErr)
			}
		}, stopCh)
		if err != nil {
			log.Printf("Error in termKeyListener: %v", err)
		}
	}()

	// Wait for the shell to exit
	err = c.Wait()
	if err != nil {
		return err
	}

	// Signal the key listener to stop after the shell exits
	close(stopCh)

	return nil
}
