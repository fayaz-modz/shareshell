package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/google/btree"
	"github.com/gorilla/websocket"
)

func handleConnect(w http.ResponseWriter, r *http.Request) {
	headers := r.Header

	otp := headers.Get("otp")
	pass := headers.Get("pass")
	if otp == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	messagesMu.Lock()
	cMsg := getMessage(otp)
	messagesMu.Unlock()

	if cMsg == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	cMsg.(*Message).Mutex.Lock()
	if cMsg.(*Message).client {
		w.WriteHeader(http.StatusBadRequest)
		cMsg.(*Message).Mutex.Unlock()
		return
	}
	cMsg.(*Message).Mutex.Unlock()

	if cMsg.(*Message).Pass != pass {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	ws, upErr := upgrader.Upgrade(w, r, nil)

	cMsg.(*Message).Mutex.Lock()
	cMsg.(*Message).client = true
	cMsg.(*Message).Mutex.Unlock()

	if upErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		cMsg.(*Message).Mutex.Lock()
		cMsg.(*Message).client = false
		cMsg.(*Message).Mutex.Unlock()
		return
	}

	newJwt, jwtErr := EncodeOTP(otp, "answer")
	if jwtErr != nil {
		ws.WriteJSON(ErrorMessage{Error: jwtErr.Error()})
		cMsg.(*Message).Mutex.Lock()
		cMsg.(*Message).client = false
		cMsg.(*Message).Mutex.Unlock()
		return
	}

	cMsg.(*Message).Mutex.Lock()
	offersdp := cMsg.(*Message).OfferSDP
	cMsg.(*Message).Mutex.Unlock()

	if sendErr := ws.WriteJSON(Message{
    Auth: newJwt,
    OfferSDP: offersdp,
  }); sendErr != nil {
		ws.Close()
		cMsg.(*Message).Mutex.Lock()
		cMsg.(*Message).client = false
		cMsg.(*Message).Mutex.Unlock()
		return
	}

	var wg sync.Mutex

	go handleAnswerConnection(ws, &wg, cMsg)
	go handleAnswerPing(ws, cMsg.(*Message), &wg)
}

func handleAnswerConnection(ws *websocket.Conn, wg *sync.Mutex, cMsg btree.Item) {
	defer func() {
		wg.Lock()
		ws.Close()
		wg.Unlock()

		cMsg.(*Message).Mutex.Lock()
		cMsg.(*Message).client = false
		cMsg.(*Message).Mutex.Unlock()
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
		if connType != "answer" {
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
			ws.WriteJSON(struct {
				Error string `json:"error"`
			}{Error: "Invalid OTP. Maybe the connection is closed."})
			wg.Unlock()
			break
		}

		if msgN.AnswerSDP != "" {
			messagesMu.Lock()
			message.(*Message).AnswerSDP = msgN.AnswerSDP
			messagesMu.Unlock()
			message.(*Message).pingOffer <- SDP
		} else if msgN.AnswerCandidates != nil {
			messagesMu.Lock()
			message.(*Message).AnswerCandidates = msgN.AnswerCandidates
			messagesMu.Unlock()
			message.(*Message).pingOffer <- Candidates
		} else {
			wg.Lock()
			tErr := ws.WriteJSON(&Message{
				OfferSDP:        message.(*Message).OfferSDP,
				OfferCandidates: message.(*Message).OfferCandidates,
			})
			wg.Unlock()
			if tErr != nil {
				fmt.Println(tErr)
				break
			}
		}
	}
}

func handleAnswerPing(ws *websocket.Conn, msg *Message, wg *sync.Mutex) {
	defer func() {
		wg.Lock()
		ws.Close()
		wg.Unlock()

		msg.Mutex.Lock()
		msg.client = false
		msg.Mutex.Unlock()
	}()

	for {
		pingAnswer := <-msg.pingAnswer
		if pingAnswer == SDP {
      fmt.Println("answer is pinged")
			wg.Lock()
			tErr := ws.WriteJSON(&Message{
				OfferSDP:        msg.OfferSDP,
				OfferCandidates: msg.OfferCandidates,
			})
			wg.Unlock()
			if tErr != nil {
				break
			}
		} else if pingAnswer == Candidates {
			wg.Lock()
			tErr := ws.WriteJSON(&Message{
				AnswerCandidates: msg.AnswerCandidates,
			})
			wg.Unlock()
			if tErr != nil {
				break
			}
		} else if pingAnswer == Close {
			break
		}
	}
}
