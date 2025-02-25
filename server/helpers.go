package main

import (
	"github.com/google/btree"
)

func getMessage(otp string) btree.Item {
  messagesMu.Lock()
  defer messagesMu.Unlock()
  return messages.Get(&Message{otp: otp})
}
