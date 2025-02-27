package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"sharec/answer"
	"sharec/offer"

	"github.com/creack/pty"
	"github.com/pion/webrtc/v4"
	"golang.org/x/term"
)

const usage = `sharec
Usage: sharec <command> [arguments]

Commands:
  help, -h, --help  to get this menu
  offer             to create a new offer
  connect <id>      to connect to an offer
`

func parse(command string) {
	if command == "help" || command == "--help" || command == "-h" {
		fmt.Print(usage)
		return
	}
	if command == "offer" {
    var url string
    if len(os.Args) > 2 {
      url = os.Args[2]
    } else {
      url = "ws://localhost:8080/offer"
    }
		cmd := exec.Command("bash")
		ptmx, err := pty.Start(cmd)
		if err != nil {
			log.Fatal("Failed to start PTY:", err)
		}
		defer ptmx.Close()
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatal("Failed to enter raw mode:", err)
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		offer.NewConn(url, func(dc *webrtc.DataChannel, peerConnection *webrtc.PeerConnection) {
			resizeChan := make(chan os.Signal, 1)
			signal.Notify(resizeChan, syscall.SIGWINCH)

			var remoteWidth, remoteHeight uint16
			go func() {
				for range resizeChan {
					localWidth, localHeight, err := term.GetSize(int(os.Stdin.Fd()))
					if err != nil {
						log.Print("Error getting local size:", err)
						continue
					}

					finalWidth := localWidth
					finalHeight := localHeight

					if remoteWidth > 0 && remoteHeight > 0 {
						if int(remoteWidth) < localWidth {
							finalWidth = int(remoteWidth)
						}
						if int(remoteHeight) < localHeight {
							finalHeight = int(remoteHeight)
						}
					}

					pty.Setsize(ptmx, &pty.Winsize{
						Rows: uint16(finalHeight),
						Cols: uint16(finalWidth),
					})

					msg := make([]byte, 5)
					msg[0] = 0x01
					binary.BigEndian.PutUint16(msg[1:3], uint16(finalHeight))
					binary.BigEndian.PutUint16(msg[3:5], uint16(finalWidth))
					dc.Send(msg)
				}
			}()
			resizeChan <- syscall.SIGWINCH

			go func() {
				buf := make([]byte, 1024)
				fmt.Print("staritng to read input")
				for {
					n, err := ptmx.Read(buf)
					if err != nil {
						log.Print("PTY read error:", err)
            peerConnection.Close()
						return
					}
					dc.Send(buf[:n])
					os.Stdout.Write(buf[:n])
				}
			}()

			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := os.Stdin.Read(buf)
					if err != nil {
						log.Print("Stdin read error:", err)
            peerConnection.Close()
						return
					}
					ptmx.Write(buf[:n])
				}
			}()

			dc.OnMessage(func(data webrtc.DataChannelMessage) {
				if len(data.Data) > 0 && data.Data[0] == 0x01 {
					remoteHeight = binary.BigEndian.Uint16(data.Data[1:3])
					remoteWidth = binary.BigEndian.Uint16(data.Data[3:5])
					resizeChan <- syscall.SIGWINCH
				} else {
					ptmx.Write(data.Data)
				}
			})

		})
		return
	}
	if command == "connect" {
		if len(os.Args) < 3 {
			fmt.Println("you need to provide an offer id. sharec connect <id>")
			return
		}
		fmt.Println("connecting to otp:", os.Args[2])
    var url string
    if len(os.Args) > 3 {
      url = os.Args[3]
    } else {
      url = "ws://localhost:8080/connect"
    }
    println("connecing to ", url)
		answer.AnswerConnection(url, os.Args[2], func(dc *webrtc.DataChannel, peerConnection *webrtc.PeerConnection) {
			resizeChan := make(chan os.Signal, 1)
			signal.Notify(resizeChan, syscall.SIGWINCH)

			go func() {
				oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
				if err != nil {
					log.Fatal("Failed to enter raw mode:", err)
				}
				defer term.Restore(int(os.Stdin.Fd()), oldState)

				buf := make([]byte, 1024)
				for {
					n, err := os.Stdin.Read(buf)
					if err != nil {
						log.Print("Stdin read error:", err)
            peerConnection.Close()
						return
					}
					dc.Send(buf[:n])
				}
			}()

			dc.OnMessage(func(data webrtc.DataChannelMessage) {
				if len(data.Data) > 0 && data.Data[0] == 0x01 {
					resizeChan <- syscall.SIGWINCH
				} else {
					os.Stdout.Write(data.Data)
				}
			})

		})
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		return
	}
	command := os.Args[1]
	parse(command)
}
