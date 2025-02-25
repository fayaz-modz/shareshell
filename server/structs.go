package main

import (
	"sync"

	"github.com/google/btree"
)

type MessageType int

const (
	SDP MessageType = iota + 1
	Candidates
	Close
)

type Message struct {
  sync.Mutex
	otp              string
	Pass             string   `json:"pass,omitempty"`
	OfferSDP         string   `json:"offerSDP,omitempty"`
	AnswerSDP        string   `json:"answerSDP,omitempty"`
	OfferCandidates  []string `json:"offerCandidates,omitempty"`
	AnswerCandidates []string `json:"answerCandidates,omitempty"`
	Auth             string   `json:"auth,omitempty"`
	pingAnswer       chan MessageType
	pingOffer        chan MessageType
	client           bool
}

type ErrorMessage struct {
	Error string `json:"error"`
}

func (m *Message) Less(item btree.Item) bool {
	return m.otp < item.(*Message).otp
}
