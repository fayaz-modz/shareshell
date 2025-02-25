package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("lovely")

func EncodeOTP(otp string, connType string) (string, error) {
	claims := jwt.MapClaims{
		"otp": otp,
    "connType": connType,
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func DecodeOTP(tokenString string) (string, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return "", "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if otp, ok := claims["otp"].(string); ok {
      if exp, okk := claims["exp"].(int64); okk && time.Now().Unix() > exp {
        return "", "", fmt.Errorf("Token Expired.")
      }
      if connType, okk := claims["connType"].(string); okk {
        return otp, connType, nil
      }
			return otp, "", fmt.Errorf("Invalid Token. Connection type is not  found.")
		}
	}

	return "", "", fmt.Errorf("Invalid Token. OTP is not found.")
}
