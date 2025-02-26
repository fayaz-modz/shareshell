package models

type MessageType int8

const (
	OfferSDP MessageType = iota
	AnswerSDP
	OfferCandidates
	AnswerCandidates
	OTP
	Auth
)

type Message struct {
	Otp              string
	OfferSDP         string   `json:"offerSDP,omitempty"`
	OfferCandidates  []string `json:"offerCandidates,omitempty"`
	AnswerSDP        string   `json:"answerSDP,omitempty"`
	AnswerCandidates []string `json:"answerCandidates,omitempty"`
	Auth             string   `json:"auth,omitempty"`
  Error            string   `json:"error,omitempty"`
}
