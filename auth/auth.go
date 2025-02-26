package auth

import (
	"fmt"
	"sharec/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// returns authToken, otp, error
func GetAuthToken(ws *websocket.Conn) (string, string, error) {
	msgChan := make(chan models.Message, 1)
	errChan := make(chan error, 1)

	go func() {
		var msg models.Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			errChan <- err
			return
		}
		msgChan <- msg
	}()

	select {
	case msg := <-msgChan:
		if msg.Auth == "" {
			return "", "", fmt.Errorf("No auth token returned.")
		} else {
			token, _, err := jwt.NewParser().ParseUnverified(msg.Auth, jwt.MapClaims{})
			if err != nil {
				return "", "", fmt.Errorf("Error parsing token: %v", err)
			}
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if claims["otp"] != nil {
					return msg.Auth, claims["otp"].(string), nil
				} else {
					fmt.Println("OTP not found")
					return "", "", fmt.Errorf("OTP not found")
				}
			} else {
        return "", "", fmt.Errorf("Error parsing token: %v", err)
      }
		}
	case err := <-errChan:
		return "", "", err
	case <-time.After(5 * time.Second):
		return "", "", fmt.Errorf("message timeout.")
	}

}
