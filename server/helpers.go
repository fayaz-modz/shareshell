package main

import (
	"github.com/google/btree"
)

func getMessage(otp string) btree.Item {
  return messages.Get(&Message{otp: otp})
}
