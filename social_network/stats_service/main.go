package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	"github.com/gorilla/mux"

	"better_errors"
	pb "proto"
)

var (
	db driver.Conn
)

type TAggregatedStats struct {
	PostId uint64 `json:"postId"`
	Views  uint64 `json:"views"`
	Likes  uint64 `json:"likes"`
}

func main() {
	ConnectClickhouseDB()

	CreateTable()

	r := mux.NewRouter()
	r.HandleFunc("/health", HealthCheckHandler).Methods("GET")
	r.HandleFunc("/cheat/{post_id}", GetPostStatsHandler).Methods("GET")

	go ConsumeKafka()
	log.Println("Starting stats service on port 8001")
	log.Panicln(http.ListenAndServe(":8001", r))
}

func ConnectClickhouseDB() {
	var err error
	db, err = clickhouse.Open(&clickhouse.Options{
		Addr: []string{"clickhouse:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
		},
	})
	better_errors.CheckErrorFatal(err, "failed to open clickhouse connection")
}

func CreateTable() {
	err := db.Exec(context.Background(), "DROP TABLE IF EXISTS post_stats")
	better_errors.CheckErrorFatal(err, "error on dropping table")

	err = db.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS post_stats (
            post_id UInt64,
            viewed UInt64,
            liked UInt64,
            timestamp DateTime
        ) engine = MergeTree()
        ORDER BY post_id
        PRIMARY KEY post_id
    `)
	better_errors.CheckErrorFatal(err, "error on creating table")
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(context.Background()); err != nil {
		http.Error(w, "Clickhouse ping error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("All good.\n"))
}

func GetPostStatsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postIdStr, ok := vars["post_id"]
	if better_errors.CheckCustomHttp(!ok, w, http.StatusBadRequest, "invalid post_id") {
		return
	}
	postId, err := strconv.ParseUint(postIdStr, 10, 64)
	if better_errors.CheckHttpError(err, w, http.StatusBadRequest, "invalid post_id value %v", postIdStr) {
		return
	}

	row := db.QueryRow(r.Context(), fmt.Sprintf(`
		SELECT
			first_value(post_id) as PostId,
			sum(viewed) as Views,
			sum(liked) as Likes
		FROM post_stats
		WHERE post_id == %v
	`, postId))
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to aggregate row") {
		return
	}

	stats := &TAggregatedStats{}
	err = row.ScanStruct(stats)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to scan stats into a struct") {
		return
	}
	response, err := json.Marshal(stats)
	if better_errors.CheckHttpError(err, w, http.StatusInternalServerError, "failed to marshal response") {
		return
	}

	w.Write(response)
}

func ConsumeKafka() {
	consumer, err := sarama.NewConsumer([]string{"kafka:9092"}, nil)
	better_errors.CheckErrorFatal(err, "failed to create kafka consumer")
	defer func() {
		better_errors.CheckErrorFatal(consumer.Close(), "failed to close kafka consumer")
	}()

	partitionConsumer, err := consumer.ConsumePartition("StatsTopic", 0, sarama.OffsetNewest)
	better_errors.CheckErrorPanic(err, "failed to create a kafka partition consumer")
	defer func() {
		better_errors.CheckErrorFatal(partitionConsumer.Close(), "failed to close kafka partition consumer")
	}()

	consumed := 0
	for {
		msg := <-partitionConsumer.Messages()
		log.Printf("Consumed message offset %d\n", msg.Offset)
		consumed++

		var statsUpdate pb.TPostStats
		err = proto.Unmarshal(msg.Value, &statsUpdate)
		better_errors.CheckErrorPanic(err, "failed to unmarshal kafka message")

		InsertIntoClickhouse(&statsUpdate)
	}
}

func InsertIntoClickhouse(info *pb.TPostStats) {
	err := db.AsyncInsert(
		context.Background(),
		fmt.Sprintf(`INSERT INTO post_stats VALUES (%v, %v, %v, now())`, info.PostId, info.Viewed, info.Liked),
		false,
	)
	better_errors.CheckErrorPanic(err, "error on executing insert query")
}
