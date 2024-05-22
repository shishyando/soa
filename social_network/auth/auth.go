package auth

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TAuthHandler struct {
	JwtPrivate *rsa.PrivateKey
	JwtPublic  *rsa.PublicKey
}

func GenerateToken(username string, password string, privateKey *rsa.PrivateKey) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":      "AuthService",
		"username": username,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	})
	return token.SignedString(privateKey)
}

func ParseToken(tokenString string, publicKey *rsa.PublicKey) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return "", fmt.Errorf("error while parsing a token: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("error: Invlid claims")
	}
	username := claims["username"].(string)
	return username, nil
}

func VerifyToken(r *http.Request, authHandler *TAuthHandler) (string, error) {
	cookie, err := r.Cookie("jwt")
	if err != nil {
		if err == http.ErrNoCookie {
			return "", fmt.Errorf("no cookie provided: %v", err.Error())
		}
		return "", fmt.Errorf("some jwt cookie error when trying to get it: %v", err.Error())
	}

	login, err := ParseToken(cookie.Value, authHandler.JwtPublic)
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %v", err.Error())
	}
	return login, nil
}

func SetCookie(login string, password string, authHandler *TAuthHandler, w http.ResponseWriter) error {
	tokenString, err := GenerateToken(login, password, authHandler.JwtPrivate)
	if err != nil {
		return fmt.Errorf("failed to sign the jwt token: %v", err.Error())
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "jwt",
		Value:   tokenString,
		Path:    "/",
		Expires: time.Now().Add(time.Hour * 72),
	})
	return nil
}

func NewAuthHandler(jwtprivateFile string, jwtPublicFile string) (*TAuthHandler, error) {
	private, err := os.ReadFile(jwtprivateFile)
	if err != nil {
		return nil, err
	}
	public, err := os.ReadFile(jwtPublicFile)
	if err != nil {
		return nil, err
	}
	jwtPrivate, err := jwt.ParseRSAPrivateKeyFromPEM(private)
	if err != nil {
		return nil, err
	}
	jwtPublic, err := jwt.ParseRSAPublicKeyFromPEM(public)
	if err != nil {
		return nil, err
	}
	return &TAuthHandler{
		JwtPrivate: jwtPrivate,
		JwtPublic:  jwtPublic,
	}, nil
}
