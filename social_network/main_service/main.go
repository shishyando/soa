package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/IBM/sarama"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"auth"
	"better_errors"
	pb "proto"
)

var (
	redisClient        *redis.Client
	postServiceClient  pb.PostServiceClient
	statsServiceClient pb.StatsServiceClient
	authHandler        *auth.TAuthHandler
	kafkaProducer      sarama.AsyncProducer
)

type TUser struct {
	Login       string `json:"login"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	Surname     string `json:"surname"`
	DateOfBirth string `json:"dateOfBirth"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var u TUser

	err := json.NewDecoder(r.Body).Decode(&u)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid input data") {
		return
	}
	exists, err := redisClient.Exists(r.Context(), u.Login).Result()
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to check that user exists") {
		return
	}
	if failed := better_errors.CheckCustomHttp(exists != 0, w, http.StatusConflict, "user %v already exists", u.Login); failed {
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	u.Password = string(hashedPassword)
	jsonUser, _ := json.Marshal(u)

	err = redisClient.Set(r.Context(), u.Login, jsonUser, 0).Err()
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "registration failed") {
		return
	}

	err = auth.SetCookie(u.Login, u.Password, authHandler, w)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to set cookie") {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var u TUser
	err := json.NewDecoder(r.Body).Decode(&u)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid request") {
		return
	}

	jsonUser, err := redisClient.Get(r.Context(), u.Login).Result()
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid login or password") {
		return
	}

	var userInDB TUser
	json.Unmarshal([]byte(jsonUser), &userInDB)

	err = bcrypt.CompareHashAndPassword([]byte(userInDB.Password), []byte(u.Password))
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid login or password") {
		return
	}

	err = auth.SetCookie(u.Login, u.Password, authHandler, w)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "Failed to set cookie") {
		return
	}
	w.WriteHeader(http.StatusOK)
	log.Printf("Login successful:\n%v", userInDB)
}

func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	login, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}

	// Decode user data
	var u TUser
	err = json.NewDecoder(r.Body).Decode(&u)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "bad request") {
		return
	}
	// Check that the token matches the user
	// user can not change it's login
	if u.Login != login {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check that the user actually exists in the database
	jsonUserInDB, err := redisClient.Get(r.Context(), u.Login).Result()
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "failed to get user's data") {
		return
	}

	// Check that the user does not try to change the password
	var userInDB TUser
	json.Unmarshal([]byte(jsonUserInDB), &userInDB)

	err = bcrypt.CompareHashAndPassword([]byte(userInDB.Password), []byte(u.Password))
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid password") {
		return
	}

	// Update DB
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	u.Password = string(hashedPassword)
	jsonUser, _ := json.Marshal(u)
	err = redisClient.Set(r.Context(), login, jsonUser, 0).Err()
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to update user's data") {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	login, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}

	// Create post
	pbReq := pb.TCreatePostRequest{}
	body, _ := io.ReadAll(r.Body)
	err = protojson.Unmarshal(body, &pbReq)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "bad request") {
		return
	}

	pbReq.AuthorLogin = login // set login from token
	pbRes, err := postServiceClient.CreatePost(r.Context(), &pbReq)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to create post") {
		return
	}
	_, err = statsServiceClient.AddPost(r.Context(), &pb.TAddPostRequest{PostId: *pbRes.PostId, AuthorLogin: login})
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to add post") {
		return
	}
	resBody, err := protojson.Marshal(pbRes)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to marshal response") {
		return
	}
	_, err = w.Write(resBody)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to respond properly") {
		return
	}
}

func UpdatePostHandler(w http.ResponseWriter, r *http.Request) {
	login, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}

	// Update post
	pbReq := pb.TUpdatePostRequest{}
	body, _ := io.ReadAll(r.Body)
	err = protojson.Unmarshal(body, &pbReq)

	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "bad request") {
		return
	}
	pbReq.AuthorLogin = login // set login from token

	_, err = postServiceClient.UpdatePost(r.Context(), &pbReq)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "failed to update post") {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	login, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}

	// Delete post
	pbReq := pb.TDeletePostRequest{}
	pbReq.AuthorLogin = login // set login from token
	pbReq.PostId, err = parsePostId(r)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid post id") {
		return
	}

	_, err = postServiceClient.DeletePost(r.Context(), &pbReq)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "failed to delete post") {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func GetPostByIdHandler(w http.ResponseWriter, r *http.Request) {
	_, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}

	// Get post
	pbReq := pb.TGetPostByIdRequest{}
	pbReq.PostId, err = parsePostId(r)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid post id") {
		return
	}

	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "failed to delete post") {
		return
	}
	pbRes, err := postServiceClient.GetPostById(r.Context(), &pbReq)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to process request") {
		return
	}
	resBody, err := protojson.Marshal(pbRes)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to marshal response") {
		return
	}
	_, err = w.Write(resBody)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to respond properly") {
		return
	}
}

func GetPostsOnPageHandler(w http.ResponseWriter, r *http.Request) {
	_, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}

	// Get posts on page
	pbReq := pb.TGetPostsOnPageRequest{}
	vars := mux.Vars(r)
	pageIdStr, ok := vars["page_id"]
	if better_errors.CheckCustomHttp(!ok, w, http.StatusBadRequest, "invalid page_id") {
		return
	}
	pbReq.PageId, err = strconv.ParseUint(pageIdStr, 10, 64)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid page_id value %v", pageIdStr) {
		return
	}

	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "failed to delete post") {
		return
	}

	pbRes, err := postServiceClient.GetPostsOnPage(r.Context(), &pbReq)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to process request") {
		return
	}
	resBody, err := protojson.Marshal(pbRes)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to marshal response") {
		return
	}
	_, err = w.Write(resBody)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to respond properly") {
		return
	}
}

func parsePostId(r *http.Request) (uint64, error) {
	vars := mux.Vars(r)
	postIdStr, ok := vars["post_id"]
	if !ok {
		return 0, fmt.Errorf("expected uint64, but `%v` found", postIdStr)
	}
	return strconv.ParseUint(postIdStr, 10, 64)
}

func ViewPostByIdHandler(w http.ResponseWriter, r *http.Request) {
	_, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}
	postId, err := parsePostId(r)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid post id") {
		return
	}
	postStats := pb.TPostStats{PostId: postId, Liked: 0, Viewed: 1}
	serializedStats, err := proto.Marshal(&postStats)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to serialize post stats message") {
		return
	}

	message := &sarama.ProducerMessage{Topic: "StatsTopic", Value: sarama.ByteEncoder(serializedStats)}

	select {
	case kafkaProducer.Input() <- message:
		w.WriteHeader(http.StatusOK)
	case err := <-kafkaProducer.Errors():
		better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to produce stats message")
	}
}

func LikePostByIdHandler(w http.ResponseWriter, r *http.Request) {
	_, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}
	postId, err := parsePostId(r)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid post id") {
		return
	}
	postStats := pb.TPostStats{PostId: postId, Liked: 1, Viewed: 0}
	serializedStats, err := proto.Marshal(&postStats)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to serialize post stats message") {
		return
	}

	message := &sarama.ProducerMessage{Topic: "StatsTopic", Value: sarama.ByteEncoder(serializedStats)}

	select {
	case kafkaProducer.Input() <- message:
		w.WriteHeader(http.StatusOK)
	case err := <-kafkaProducer.Errors():
		better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to produce stats message")
	}
}

func PostStatsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}
	pbReq := &pb.TGetPostStatsRequest{}
	pbReq.PostId, err = parsePostId(r)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid post id") {
		return
	}
	pbRes, err := statsServiceClient.GetPostStats(r.Context(), pbReq)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to process request") {
		return
	}
	resBody, err := protojson.Marshal(pbRes)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to marshal response") {
		return
	}
	_, err = w.Write(resBody)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to respond properly") {
		return
	}
}

func TopPostsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}
	vars := mux.Vars(r)
	sortBy, ok := vars["type"]
	ok = ok && (sortBy == "likes" || sortBy == "views")
	if better_errors.CheckCustomHttp(!ok, w, http.StatusBadRequest, "Should sort by `liks` or `views`") {
		return
	}

	pbRes, err := statsServiceClient.GetTopPosts(r.Context(), &pb.TGetTopPostsRequest{OrderBy: sortBy})
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to process request") {
		return
	}
	resBody, err := protojson.Marshal(pbRes)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to marshal response") {
		return
	}
	_, err = w.Write(resBody)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to respond properly") {
		return
	}
}

func TopAuthorsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := auth.VerifyToken(r, authHandler)
	if better_errors.CheckHttpError(err, w, http.StatusUnauthorized, "invalid token") {
		return
	}

	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to process request") {
		return
	}
	pbRes, err := statsServiceClient.GetTopAuthors(r.Context(), &emptypb.Empty{})
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to process request") {
		return
	}
	resBody, err := protojson.Marshal(pbRes)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to marshal response") {
		return
	}
	_, err = w.Write(resBody)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to respond properly") {
		return
	}
}

func main() {
	privateKeyPath := flag.String("private", "", "path to JWT private key file")
	publicKeyPath := flag.String("public", "", "path to JWT public key file")
	port := flag.Int("port", 8000, "http server port")
	redisPort := flag.Int("redis_port", 6379, "redis port")
	flag.Parse()

	better_errors.CheckCustomFatal(port == nil, "invalid port")
	better_errors.CheckCustomFatal(redisPort == nil, "invalid redis port")
	better_errors.CheckCustomFatal(privateKeyPath == nil || *privateKeyPath == "", "invalid private key path")
	better_errors.CheckCustomFatal(publicKeyPath == nil || *publicKeyPath == "", "invalid public key path")
	privateKeyAbsPath, err := filepath.Abs(*privateKeyPath)
	better_errors.CheckErrorFatal(err, "private key error")
	publicKeyAbsPath, err := filepath.Abs(*publicKeyPath)
	better_errors.CheckErrorFatal(err, "public key error")

	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("redis:%d", *redisPort),
		Password: "",
		DB:       0,
	})
	kafkaProducer, err = sarama.NewAsyncProducer([]string{"kafka:9092"}, nil)
	better_errors.CheckErrorFatal(err, "failed to create kafka producer")

	authHandler, err = auth.NewAuthHandler(privateKeyAbsPath, publicKeyAbsPath)
	better_errors.CheckErrorFatal(err, "failed to create auth handler")

	grpcConnPosts, err := grpc.Dial("post_service:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	better_errors.CheckErrorFatal(err, "failed to dial")
	defer grpcConnPosts.Close()
	postServiceClient = pb.NewPostServiceClient(grpcConnPosts)

	grpcConnStats, err := grpc.Dial("stats_service:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	better_errors.CheckErrorFatal(err, "failed to dial")
	defer grpcConnStats.Close()
	statsServiceClient = pb.NewStatsServiceClient(grpcConnStats)

	r := mux.NewRouter()
	r.HandleFunc("/users/register", RegisterHandler).Methods("POST")
	r.HandleFunc("/users/login", LoginHandler).Methods("POST")
	r.HandleFunc("/users", UpdateUserHandler).Methods("PUT")
	r.HandleFunc("/posts/create", CreatePostHandler).Methods("POST")
	r.HandleFunc("/posts/update", UpdatePostHandler).Methods("PUT")
	r.HandleFunc("/posts/delete/{post_id}", DeletePostHandler).Methods("DELETE")
	r.HandleFunc("/posts/single/{post_id}", GetPostByIdHandler).Methods("GET")
	r.HandleFunc("/posts/page/{page_id}", GetPostsOnPageHandler).Methods("GET")
	r.HandleFunc("/posts/viewed/{post_id}", ViewPostByIdHandler).Methods("PUT")
	r.HandleFunc("/posts/liked/{post_id}", LikePostByIdHandler).Methods("PUT")
	r.HandleFunc("/posts/stats/{post_id}", PostStatsHandler).Methods("GET")
	r.HandleFunc("/posts/top/{type}", TopPostsHandler).Methods("GET")
	r.HandleFunc("/users/top", TopAuthorsHandler).Methods("GET")

	log.Printf("Staring main user server on port %d", *port)

	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), r)
	better_errors.CheckErrorFatal(err, "failed to serve")
}
