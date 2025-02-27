package answer

import (
	"fmt"
	"net/http"
	"sharec/auth"
	"sharec/models"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

var Msg = models.Message{
	Otp:              "",
	OfferSDP:         "",
	AnswerSDP:        "",
	OfferCandidates:  make([]string, 0),
	AnswerCandidates: make([]string, 0),
	Auth:             "",
}

var MsgMu sync.Mutex

type DataChannel = webrtc.DataChannel
type DataChannelMessage = webrtc.DataChannelMessage

func AnswerConnection(wsURL string, otp string, onConnectionOpen func(dataChannel *DataChannel, peerConnection *webrtc.PeerConnection)) {
	isConnectedMu := &sync.Mutex{}
	isConnected := false
	Msg.Otp = otp

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := peerConnection.Close(); err != nil {
			fmt.Printf("cannot close peerConnection: %v\n", err)
		}
	}()

	reqHeaders := http.Header{}
	reqHeaders.Add("otp", otp)

	var wsMu sync.Mutex
	ws, res, err := websocket.DefaultDialer.Dial(wsURL, reqHeaders)
	if res.StatusCode == http.StatusNotFound {
		fmt.Println("Invalid OTP.")
		return
	}
	if err != nil {
		panic(err)
	}
	defer ws.Close()

	token, _, err := auth.GetAuthToken(ws)
	if err != nil {
		fmt.Println("could not get auth token:", err)
		return
	}
	Msg.Auth = token

	var wg sync.WaitGroup
	var remoteSDPWg sync.WaitGroup
	remoteSDPWg.Add(1)
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			var msg models.Message
			err := ws.ReadJSON(&msg)
			if err != nil {
				isConnectedMu.Lock()
				if isConnected {
					isConnectedMu.Unlock()
					break
				}
				isConnectedMu.Unlock()
				fmt.Println("could not read message:", err)
				break
			}
			if msg.Error != "" {
				fmt.Println("error: ", msg.Error)
			}
			if msg.OfferSDP != "" {
				fmt.Println("received offer sdp")
				MsgMu.Lock()
				Msg.OfferSDP = msg.OfferSDP
				err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
					SDP:  msg.OfferSDP,
					Type: webrtc.SDPTypeOffer,
				})
				if err != nil {
					fmt.Println("could not set remote description:", err)
					remoteSDPWg.Done()
					MsgMu.Unlock()
					return
				}
				MsgMu.Unlock()
				remoteSDPWg.Done()
			}
			if msg.OfferCandidates != nil {
				fmt.Println("received offer candidates")
				MsgMu.Lock()
				Msg.OfferCandidates = append(Msg.OfferCandidates, msg.OfferCandidates...)
				for _, candidate := range msg.OfferCandidates {
					err := peerConnection.AddICECandidate(webrtc.ICECandidateInit{
						Candidate: candidate,
					})
					if err != nil {
						fmt.Println("Error adding ICE candidate:", err)
					}
				}
				MsgMu.Unlock()
				fmt.Println("received offer candidates")
			}
		}
	}()

	remoteSDPWg.Wait()

	offer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	MsgMu.Lock()
	Msg.AnswerSDP = peerConnection.LocalDescription().SDP
	MsgMu.Unlock()

	wsMu.Lock()
	fmt.Println("writing answer sdp")
	err = ws.WriteJSON(models.Message{
		Auth:      Msg.Auth,
		AnswerSDP: Msg.AnswerSDP,
	})
	if err != nil {
		fmt.Println("could not write answer sdp:", err)
		return
	}
	wsMu.Unlock()

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		MsgMu.Lock()
		Msg.AnswerCandidates = append(Msg.AnswerCandidates, candidate.ToJSON().Candidate)
		MsgMu.Unlock()

		wsMu.Lock()
		fmt.Println("sending new candidates")
		err := ws.WriteJSON(models.Message{
			Auth:             Msg.Auth,
			AnswerCandidates: []string{candidate.ToJSON().Candidate},
		})
		if err != nil {
      isConnectedMu.Lock()
      if isConnected {
        isConnectedMu.Unlock()
        return
      }
      isConnectedMu.Unlock()
			fmt.Println("error sending candidates:", err)
			ws.Close()
			return
		}
		wsMu.Unlock()

	})

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", state.String())
		if state == webrtc.PeerConnectionStateConnected {
      isConnectedMu.Lock()
			isConnected = true
      isConnectedMu.Unlock()
			wg.Add(1)
			ws.Close()
		} else if state == webrtc.PeerConnectionStateFailed {
			fmt.Println("Peer Connection has gone to failed exiting")
			wg.Done()
			return
		} else if state == webrtc.PeerConnectionStateClosed {
			fmt.Println("Peer Connection has gone to closed exiting")
			wg.Done()
			return
		} else {
			wg.Done()
		}
	})

	peerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		onConnectionOpen(dataChannel, peerConnection)
	})
	wg.Wait()
}
