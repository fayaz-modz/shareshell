package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

func handleOffer(w http.ResponseWriter, r *http.Request) {
  fmt.Println("new offer connection")

	url := r.URL
	headers := r.Header

	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		fmt.Println(err)
	}

	var otp string
	if url.Path == "/offer" {
		i := 0
		for {
			if i > 10 {
				ws.WriteJSON(struct {
					Error string `json:"error"`
				}{Error: "Could not generate OTP"})
				ws.Close()
				return
			}
			otp = fmt.Sprintf("%d", rand.Intn(100000))
			messagesMu.Lock() // !! Lock
			findOtp := getMessage(otp)
			if findOtp == nil {
				//!! messageMU is still locked
				break
			}
			messagesMu.Unlock() // !! Unlock
		}
	} else {
		newOtp := url.Path
		messagesMu.Lock() // !! Lock
		findOtp := getMessage(newOtp)
		if findOtp != nil {
			messagesMu.Unlock() // !! Unlock
			ws.WriteJSON(struct {
				Error string `json:"error"`
			}{Error: "Invalid OTP"})
			ws.Close()
			return
		}
		// !! messageMu is still locked
		otp = newOtp
	}

	msg := Message{
		otp:        otp,
		Pass:       headers.Get("pass"),
		pingAnswer: make(chan MessageType),
		pingOffer:  make(chan MessageType),
	}

	messages.ReplaceOrInsert(&msg)
	messagesMu.Unlock()

	auth, jwtErr := EncodeOTP(otp, "offer")

	if jwtErr != nil {
		ws.WriteJSON(struct {
			Error string `json:"error"`
		}{Error: "Could not generate auth"})
		ws.Close()
		return
	}

  fmt.Println("offer connection validated")
	errMsg := ws.WriteJSON(Message{Auth: auth})

	if errMsg != nil {
		ws.Close()
		messagesMu.Lock()
		messages.Delete(&Message{otp: otp})
		messagesMu.Unlock()
		return
	}

	var wg sync.Mutex

	go handleOfferConnection(ws, &msg, &wg)
	go handlePingOffer(ws, &msg, &wg)
}

func handleOfferConnection(ws *websocket.Conn, msg *Message, wg *sync.Mutex) {
	defer func() {
		messagesMu.Lock()
		msgn := messages.Get(msg)
		if msgn != nil {
			messages.Delete(msg)
		}
		messagesMu.Unlock()
		wg.Lock()
		ws.Close()
		wg.Unlock()
		msg.pingAnswer <- Close
	}()

	for {
		var msgN Message
		if err := ws.ReadJSON(&msgN); err != nil {
			break
		}

		otp, connType, err := DecodeOTP(msgN.Auth)
		if err != nil {
			wg.Lock()
			ws.WriteJSON(ErrorMessage{Error: "Invalid Token"})
			wg.Unlock()
			break
		}
		if connType != "offer" {
			wg.Lock()
			ws.WriteJSON(ErrorMessage{Error: "Invalid Token. Invalid connection type."})
			wg.Unlock()
			break
		}

		messagesMu.Lock()
		message := getMessage(otp)
		messagesMu.Unlock()
		if message == nil {
			wg.Lock()
			ws.WriteJSON(ErrorMessage{Error: "Invalid OTP"})
			wg.Unlock()
			break
		}

		if msgN.OfferSDP != "" {
      fmt.Println("offer sdp recieved")
			messagesMu.Lock()
			msg.OfferSDP = msgN.OfferSDP
			messagesMu.Unlock()
			msg.pingAnswer <- SDP
		} else if msgN.OfferCandidates != nil {
      fmt.Println("offer candidates recieved")
			messagesMu.Lock()
			msg.OfferCandidates = msgN.OfferCandidates
			messagesMu.Unlock()
			msg.pingAnswer <- Candidates
		} else {
			//sendMsg := Message{
			//	AnswerSDP:        msg.AnswerSDP,
			//	AnswerCandidates: msg.AnswerCandidates,
			//}
      fmt.Println("unknown answer recieved")
			//wg.Lock()
			//tErr := ws.WriteJSON(&sendMsg)
			//wg.Unlock()
			//if tErr != nil {
			//	break
			//}
		}
	}
}

func handlePingOffer(ws *websocket.Conn, msg *Message, wg *sync.Mutex) {
	for {
		pingOffer := <-msg.pingOffer
    fmt.Println("offer is pinged. sending ", pingOffer)
		if pingOffer == SDP {
			tErr := ws.WriteJSON(Message{AnswerSDP: msg.AnswerSDP})
			if tErr != nil {
				break
			}
		} else if pingOffer == Candidates {
			wg.Lock()
			tErr := ws.WriteJSON(Message{AnswerCandidates: msg.AnswerCandidates})
			wg.Unlock()
			if tErr != nil {
				break
			}
		} else if pingOffer == Close {
      ws.Close()
      break
    }
	}
}
