package main

import (
	"crypto/rsa"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

var redisClient *redis.Client

type User struct {
	Login       string `json:"login"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	Surname     string `json:"surname"`
	DateOfBirth string `json:"dateOfBirth"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

type AuthHandlers struct {
	jwtPrivate *rsa.PrivateKey
	jwtPublic  *rsa.PublicKey
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
		fmt.Println("Error while parsing a token:", err)
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		fmt.Println("Error: Invalid claims")
		return "", fmt.Errorf("error: Invlid claims")
	}

	username := claims["username"].(string)
	return username, nil
}

func NewAuthHandlers(jwtprivateFile string, jwtPublicFile string) *AuthHandlers {
	private, err := os.ReadFile(jwtprivateFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	public, err := os.ReadFile(jwtPublicFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	jwtPrivate, err := jwt.ParseRSAPrivateKeyFromPEM(private)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	jwtPublic, err := jwt.ParseRSAPublicKeyFromPEM(public)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return &AuthHandlers{
		jwtPrivate: jwtPrivate,
		jwtPublic:  jwtPublic,
	}
}

func (h *AuthHandlers) registerHandler(w http.ResponseWriter, r *http.Request) {
	var u User
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		http.Error(w, "Invalid input data", http.StatusBadRequest)
		return
	}
	exists, err := redisClient.Exists(r.Context(), u.Login).Result()
	if err != nil {
		log.Fatalf("Failed to check that user `%v` exists, error: `%v`", u.Login, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if exists != 0 {
		log.Printf("User `%v` already exists", u.Login)
		http.Error(w, "User with this login already exists", http.StatusConflict)
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	u.Password = string(hashedPassword)
	jsonUser, _ := json.Marshal(u)

	err = redisClient.Set(r.Context(), u.Login, jsonUser, 0).Err()
	if err != nil {
		log.Fatalf("Unable to register user with login: `%v`, error: `%v`", u.Login, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set the token
	tokenString, err := GenerateToken(u.Login, u.Password, h.jwtPrivate)
	if err != nil {
		log.Fatalf("Unable to sign token: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "jwt",
		Value: tokenString,
	})
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandlers) loginHandler(w http.ResponseWriter, r *http.Request) {
	var u User
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	jsonUser, err := redisClient.Get(r.Context(), u.Login).Result()
	if err != nil {
		http.Error(w, "Invalid login or password", http.StatusBadRequest)
		log.Fatalf("Failed to get user's `%v` data, error: %v", u.Login, err)
		return
	}

	var userInDB User
	json.Unmarshal([]byte(jsonUser), &userInDB)

	err = bcrypt.CompareHashAndPassword([]byte(userInDB.Password), []byte(u.Password))
	if err != nil {
		log.Printf("Invalid login or password:\n`%v`\n`%v`\n`%v`\n`%v`\n", userInDB.Login, userInDB.Password, u.Login, u.Password)
		http.Error(w, "Invalid login or password", http.StatusBadRequest)
		return
	}

	tokenString, err := GenerateToken(u.Login, u.Password, h.jwtPrivate)
	if err != nil {
		log.Fatalf("Unable to sign token: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set the token
	http.SetCookie(w, &http.Cookie{
		Name:  "jwt",
		Value: tokenString,
	})
	w.WriteHeader(http.StatusOK)
	log.Printf("Login successful:\n%v", userInDB)
}

func (h *AuthHandlers) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	// Verify token
	cookie, err := r.Cookie("jwt")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tokenLogin, err := ParseToken(cookie.Value, h.jwtPublic)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Decode user data
	var u User
	err = json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		http.Error(w, "Invalid input data", http.StatusBadRequest)
		return
	}
	// Check that the token matches the user
	// user can not change it's login
	if u.Login != tokenLogin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check that the user actually exists in the database
	jsonUserInDB, err := redisClient.Get(r.Context(), u.Login).Result()
	if err != nil {
		http.Error(w, "No such user", http.StatusBadRequest)
		log.Fatalf("Failed to get user's `%v` data, error: %v", u.Login, err)
		return
	}

	// Check that the user does not try to change the password
	var userInDB User
	json.Unmarshal([]byte(jsonUserInDB), &userInDB)

	err = bcrypt.CompareHashAndPassword([]byte(userInDB.Password), []byte(u.Password))
	if err != nil {
		log.Printf("Possible password change attempt: %v", u.Login)
		http.Error(w, "Invalid password", http.StatusBadRequest)
		return
	}

	// Update DB
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	u.Password = string(hashedPassword)
	jsonUser, _ := json.Marshal(u)
	err = redisClient.Set(r.Context(), tokenLogin, jsonUser, 0).Err()
	if err != nil {
		log.Fatalf("Failed to update user's data: %v", u)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	privateKeyPath := flag.String("private", "", "path to JWT private key file")
	publicKeyPath := flag.String("public", "", "path to JWT public key file")
	port := flag.Int("port", 8000, "http server port")
	redisPort := flag.Int("redis_port", 6379, "redis port")
	flag.Parse()

	if port == nil {
		fmt.Fprintln(os.Stderr, "Port is required")
		os.Exit(1)
	}

	if redisPort == nil {
		fmt.Fprintln(os.Stderr, "Redis port is required")
		os.Exit(1)
	}

	if privateKeyPath == nil || *privateKeyPath == "" {
		fmt.Fprintln(os.Stderr, "Please provide a path to JWT private key file")
		os.Exit(1)
	}

	if publicKeyPath == nil || *publicKeyPath == "" {
		fmt.Fprintln(os.Stderr, "Please provide a path to JWT public key file")
		os.Exit(1)
	}

	privateKeyAbsPath, err := filepath.Abs(*privateKeyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	publicKeyAbsPath, err := filepath.Abs(*publicKeyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("redis:%d", *redisPort),
		Password: "",
		DB:       0,
	})

	authHandlers := NewAuthHandlers(privateKeyAbsPath, publicKeyAbsPath)

	r := mux.NewRouter()
	r.HandleFunc("/users/register", authHandlers.registerHandler).Methods("POST")
	r.HandleFunc("/users/login", authHandlers.loginHandler).Methods("POST")
	r.HandleFunc("/users", authHandlers.updateUserHandler).Methods("PUT")

	log.Printf("Staring main user server on port %d", *port)

	if err = http.ListenAndServe(fmt.Sprintf(":%d", *port), r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
