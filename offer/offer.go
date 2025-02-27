package offer

import (
	"fmt"
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

type ConnectionState int8

const (
	Handshaking ConnectionState = iota
	ExchangingIceCondidates
	Connected
	Disconnected
)

type DataChannelMessage = webrtc.DataChannelMessage
type DataChannel = webrtc.DataChannel

func NewConn(wsURL string, onConnectionOpen func(dataChannel *DataChannel, peerConnection *webrtc.PeerConnection)) {
	forceClose := false
	var MsgMu sync.Mutex

	var wg sync.WaitGroup
	wg.Add(1)

	var pendingCandidatesMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	peerConnection, err := webrtc.NewPeerConnection(config)

	if err != nil {
		fmt.Println("Could not create new webrtc connection: ", err)
		return
	}
	defer func() {
		if cErr := peerConnection.Close(); cErr != nil {
			fmt.Printf("cannot close peerConnection: %v\n", cErr)
		}
	}()

	wsLock := &sync.Mutex{}
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		fmt.Println("could not contact server:", err)
		return
	}
	defer ws.Close()

	auth, otp, err := auth.GetAuthToken(ws)
	if err != nil {
		fmt.Println("could not get auth token:", err)
		wg.Done()
		return
	}
	Msg.Auth = auth
	Msg.Otp = otp
	fmt.Println("access OTP: ", otp)

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		pendingCandidatesMux.Lock()
		defer pendingCandidatesMux.Unlock()

		desc := peerConnection.RemoteDescription()
		pendingCandidates = append(pendingCandidates, candidate)

		MsgMu.Lock()
		Msg.OfferCandidates = append(Msg.OfferCandidates, candidate.ToJSON().Candidate)
		MsgMu.Unlock()
		if desc != nil {
			encodeCandidates := make([]string, 0, len(pendingCandidates))
			for _, candidate := range pendingCandidates {
				encodeCandidates = append(encodeCandidates, candidate.ToJSON().Candidate)
			}
			wsLock.Lock()
			fmt.Println("sending offer candidates")
			err := ws.WriteJSON(models.Message{
				Auth:            Msg.Auth,
				OfferCandidates: encodeCandidates,
			})
			wsLock.Unlock()
			if err != nil {
				wg.Done()
				return
			}
			fmt.Println("sent offer candidates")
			pendingCandidates = make([]*webrtc.ICECandidate, 0)
		}
	})

  ordered := true
  maxRetransmits := uint16(0)
	dataChannel, err := peerConnection.CreateDataChannel("data", &webrtc.DataChannelInit{
    Ordered: &ordered,
    MaxRetransmits: &maxRetransmits,
	})
	if err != nil {
		fmt.Println("data channel creation failed.", err)
	}

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		fmt.Println("Could not create webrtc offer:", err)
		return
	}
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		fmt.Println("Filed to set local description: ", err)
		return
	}
	Msg.OfferSDP = peerConnection.LocalDescription().SDP

	wsLock.Lock()
	fmt.Println("sending offer sdp")
	if err := ws.WriteJSON(struct {
		Auth     string `json:"auth"`
		OfferSDP string `json:"offerSDP"`
	}{
		Auth:     auth,
		OfferSDP: Msg.OfferSDP,
	}); err != nil {
		fmt.Println("could not send offer sdp:", err)
		wg.Done()
		return
	}
	wsLock.Unlock()

	go func() {
		defer wg.Done()
		for {
			var msg models.Message
			err := ws.ReadJSON(&msg)
			if err != nil {
				if forceClose {
					break
				}
				fmt.Println("connection closed unexpectedly.", err)
				break
			}
			if msg.Error != "" {
				fmt.Println("error: ", msg.Error)
				break
			}

			if msg.AnswerCandidates != nil {
				fmt.Println("received answer candidates")
				MsgMu.Lock()
				for _, candidate := range msg.AnswerCandidates {
					Msg.AnswerCandidates = append(Msg.AnswerCandidates, candidate)
					peerConnection.AddICECandidate(webrtc.ICECandidateInit{
						Candidate: candidate,
					})
				}
				MsgMu.Unlock()
			}
			if msg.AnswerSDP != "" {
				fmt.Println("received answer SDP")
				pendingCandidatesMux.Lock()
				MsgMu.Lock()
				Msg.AnswerSDP = msg.AnswerSDP
				Msg.AnswerCandidates = []string{}
				peerConnection.SetRemoteDescription(webrtc.SessionDescription{
					Type: webrtc.SDPTypeAnswer,
					SDP:  Msg.AnswerSDP,
				})
				MsgMu.Unlock()
				pendingCandidatesMux.Unlock()
			}
		}
	}()

	peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", state.String())

		if state == webrtc.PeerConnectionStateConnected {
			forceClose = true
			wg.Add(1)
			ws.Close()
		}

		if state == webrtc.PeerConnectionStateFailed {
			fmt.Println("Peer Connection has gone to failed exiting")
			forceClose = true
      err := ws.Close()
      if err != nil {
        wg.Done()
      }
		}

		if state == webrtc.PeerConnectionStateClosed {
			fmt.Println("Peer Connection has gone to closed exiting")
			forceClose = true
      if err := ws.Close(); err != nil {
        wg.Done()
      }
		}
	})

	dataChannel.OnOpen(func() {
		onConnectionOpen(dataChannel, peerConnection)
	})

	wg.Wait()
}
