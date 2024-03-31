package main

import (
	"crypto/rsa"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "proto"
)

var redisClient *redis.Client
var postServiceClient pb.PostServiceClient

type TUser struct {
	Login       string `json:"login"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	Surname     string `json:"surname"`
	DateOfBirth string `json:"dateOfBirth"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

type TAuthHandlers struct {
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

func NewAuthHandlers(jwtprivateFile string, jwtPublicFile string) *TAuthHandlers {
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
	return &TAuthHandlers{
		JwtPrivate: jwtPrivate,
		JwtPublic:  jwtPublic,
	}
}

func (h *TAuthHandlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var u TUser
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		http.Error(w, "Invalid input data", http.StatusBadRequest)
		return
	}
	exists, err := redisClient.Exists(r.Context(), u.Login).Result()
	if err != nil {
		log.Printf("Failed to check that user `%v` exists, error: `%v`", u.Login, err)
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
		log.Printf("Unable to register user with login: `%v`, error: `%v`", u.Login, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set the token
	tokenString, err := GenerateToken(u.Login, u.Password, h.JwtPrivate)
	if err != nil {
		log.Printf("Unable to sign token: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  "jwt",
		Value: tokenString,
	})
	w.WriteHeader(http.StatusOK)
}

func (h *TAuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var u TUser
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	jsonUser, err := redisClient.Get(r.Context(), u.Login).Result()
	if err != nil {
		http.Error(w, "Invalid login or password", http.StatusBadRequest)
		// time.Sleep(time.Second * 3)
		log.Printf("Failed to get user's `%v` data, error: %v", u.Login, err)
		return
	}

	var userInDB TUser
	json.Unmarshal([]byte(jsonUser), &userInDB)

	err = bcrypt.CompareHashAndPassword([]byte(userInDB.Password), []byte(u.Password))
	if err != nil {
		log.Printf("Invalid login or password:\n`%v`\n`%v`\n`%v`\n`%v`\n", userInDB.Login, userInDB.Password, u.Login, u.Password)
		http.Error(w, "Invalid login or password", http.StatusBadRequest)
		return
	}

	tokenString, err := GenerateToken(u.Login, u.Password, h.JwtPrivate)
	if err != nil {
		log.Printf("Unable to sign token: %v\n", err)
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

func (h *TAuthHandlers) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	// Verify token
	cookie, err := r.Cookie("jwt")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("No cookie provided: %v", err.Error())
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Some jwt cookie error when trying to get it: %v", err.Error())
		return
	}

	tokenLogin, err := ParseToken(cookie.Value, h.JwtPublic)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Printf("ParseToken failed: %v", err.Error())
		return
	}

	// Decode user data
	var u TUser
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
		log.Printf("Failed to get user's `%v` data, error: %v", u.Login, err)
		http.Error(w, "No such user", http.StatusBadRequest)
		return
	}

	// Check that the user does not try to change the password
	var userInDB TUser
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
		log.Printf("Failed to update user's data: %v", u)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TAuthHandlers) CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	// Verify token
	cookie, err := r.Cookie("jwt")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("No cookie provided: %v", err.Error())
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Some jwt cookie error when trying to get it: %v", err.Error())
		return
	}

	login, err := ParseToken(cookie.Value, h.JwtPublic)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Printf("ParseToken failed: %v", err.Error())
		return
	}

	// Create post
	qwe, _ := httputil.DumpRequest(r, true)
	log.Println(qwe)
	pbReq := &pb.TCreatePostRequest{}
	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(r.Body, pbReq)

	pbReq.AuthorLogin = login // set login from token

	if err != nil {
		http.Error(w, "Invalid input data", http.StatusBadRequest)
		log.Printf("Invalid json to proto %v", err.Error())
		return
	}

	pbRes, err := postServiceClient.CreatePost(r.Context(), pbReq)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Create post failed: %v", err.Error())
		return
	}
	if !pbRes.Created {
		http.Error(w, "Post not created", http.StatusInternalServerError)
		log.Printf("Post not created: %v", pbRes.String())
		return
	}

	data, err := json.Marshal(pbRes)
	if err != nil {
		http.Error(w, "Marshal error", http.StatusInternalServerError)
		log.Printf("Create post marshal response failed: %v", err.Error())
		return
	}

	w.Write(data)
}

func (h *TAuthHandlers) UpdatePostHandler(w http.ResponseWriter, r *http.Request) {
	// Verify token
	cookie, err := r.Cookie("jwt")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("No cookie provided: %v", err.Error())
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Some jwt cookie error when trying to get it: %v", err.Error())
		return
	}

	login, err := ParseToken(cookie.Value, h.JwtPublic)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Printf("ParseToken failed: %v", err.Error())
		return
	}

	// Update post
	pbReq := pb.TUpdatePostRequest{}
	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(r.Body, &pbReq)

	if err != nil {
		http.Error(w, "Invalid input data", http.StatusBadRequest)
		log.Printf("Invalid json to proto %v", err.Error())
		return
	}
	pbReq.AuthorLogin = login // set login from token

	pbRes, err := postServiceClient.UpdatePost(r.Context(), &pbReq)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Update post failed: %v", err.Error())
		return
	}
	if !pbRes.Updated {
		http.Error(w, "Post not updated", http.StatusInternalServerError)
		log.Printf("Post not updated: %v", pbRes.String())
		return
	}

	data, err := json.Marshal(pbRes)
	if err != nil {
		http.Error(w, "Marshal error", http.StatusInternalServerError)
		log.Printf("Update post marshal response failed")
		return
	}

	w.Write(data)
}

func (h *TAuthHandlers) DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	// Verify token
	cookie, err := r.Cookie("jwt")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("No cookie provided: %v", err.Error())
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Some jwt cookie error when trying to get it: %v", err.Error())
		return
	}

	login, err := ParseToken(cookie.Value, h.JwtPublic)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Printf("ParseToken failed: %v", err.Error())
		return
	}

	// Delete post
	pbReq := pb.TDeletePostRequest{}
	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(r.Body, &pbReq)

	if err != nil {
		http.Error(w, "Invalid input data", http.StatusBadRequest)
		log.Printf("Invalid json to proto %v", err.Error())
		return
	}
	pbReq.AuthorLogin = login // set login from token

	pbRes, err := postServiceClient.DeletePost(r.Context(), &pbReq)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Delete post failed: %v", err.Error())
		return
	}
	if !pbRes.Deleted {
		http.Error(w, "Post not deleted", http.StatusInternalServerError)
		log.Printf("Post not deleted: %v", pbRes.String())
		return
	}

	data, err := json.Marshal(pbRes)
	if err != nil {
		http.Error(w, "Marshal error", http.StatusInternalServerError)
		log.Printf("Delete post marshal response failed")
		return
	}

	w.Write(data)
}

func (h *TAuthHandlers) GetPostByIdHandler(w http.ResponseWriter, r *http.Request) {
	// Verify token
	cookie, err := r.Cookie("jwt")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("No cookie provided: %v", err.Error())
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Some jwt cookie error when trying to get it: %v", err.Error())
		return
	}

	_, err = ParseToken(cookie.Value, h.JwtPublic)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Printf("ParseToken failed: %v", err.Error())
		return
	}

	// Delete post
	pbReq := pb.TGetPostByIdRequest{}
	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(r.Body, &pbReq)

	if err != nil {
		http.Error(w, "Invalid input data", http.StatusBadRequest)
		log.Printf("Invalid json to proto %v", err.Error())
		return
	}

	pbRes, err := postServiceClient.GetPostById(r.Context(), &pbReq)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Get post failed: %v", err.Error())
		return
	}

	data, err := json.Marshal(pbRes)
	if err != nil {
		http.Error(w, "Marshal error", http.StatusInternalServerError)
		log.Printf("Get post by id post marshal response failed")
		return
	}

	w.Write(data)
}

func (h *TAuthHandlers) GetPostsOnPageHandler(w http.ResponseWriter, r *http.Request) {
	// Verify token
	cookie, err := r.Cookie("jwt")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("No cookie provided: %v", err.Error())
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("Some jwt cookie error when trying to get it: %v", err.Error())
		return
	}

	_, err = ParseToken(cookie.Value, h.JwtPublic)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Printf("ParseToken failed: %v", err.Error())
		return
	}

	// Delete post
	pbReq := pb.TGetPostsOnPageRequest{}
	unmarshaller := &jsonpb.Unmarshaler{}
	err = unmarshaller.Unmarshal(r.Body, &pbReq)

	if err != nil {
		http.Error(w, "Invalid input data", http.StatusBadRequest)
		log.Printf("Invalid json to proto %v", err.Error())
		return
	}

	pbRes, err := postServiceClient.GetPostsOnPage(r.Context(), &pbReq)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(pbRes)
	if err != nil {
		http.Error(w, "Marshal error", http.StatusInternalServerError)
		log.Printf("Get posts on page marshal response failed")
		return
	}

	w.Write(data)
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
	grpcconn, err := grpc.Dial("post_service:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer grpcconn.Close()
	postServiceClient = pb.NewPostServiceClient(grpcconn)

	r := mux.NewRouter()
	r.HandleFunc("/users/register", authHandlers.RegisterHandler).Methods("POST")
	r.HandleFunc("/users/login", authHandlers.LoginHandler).Methods("POST")
	r.HandleFunc("/users", authHandlers.UpdateUserHandler).Methods("PUT")
	r.HandleFunc("/posts/create", authHandlers.CreatePostHandler).Methods("PUT")
	r.HandleFunc("/posts/update", authHandlers.UpdatePostHandler).Methods("PUT")
	r.HandleFunc("/posts/delete", authHandlers.DeletePostHandler).Methods("DELETE")
	r.HandleFunc("/posts/single", authHandlers.GetPostByIdHandler).Methods("GET")
	r.HandleFunc("/posts/page", authHandlers.GetPostsOnPageHandler).Methods("GET")

	log.Printf("Staring main user server on port %d", *port)

	if err = http.ListenAndServe(fmt.Sprintf(":%d", *port), r); err != nil {
		log.Fatalf(err.Error())
	}
}
