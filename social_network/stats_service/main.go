package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/IBM/sarama"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/gorilla/mux"

	"better_errors"
	pb "proto"
)

var (
	db driver.Conn
)

type server struct {
	pb.UnimplementedStatsServiceServer
}

func main() {
	ConnectClickhouseDB()

	CreateTable()

	listenAddress := ":50051"
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Printf("Failed to listen: %v", err)
		os.Exit(1)
	}

	serverInstance := grpc.NewServer()
	pb.RegisterStatsServiceServer(serverInstance, &server{})

	r := mux.NewRouter()
	r.HandleFunc("/health", HealthCheckHandler).Methods("GET")
	r.HandleFunc("/cheat/{post_id}", GetPostStatsHandler).Methods("GET")

	go ConsumeKafka()
	go func() {
		log.Println("Starting stats service on port 8001")
		log.Panicln(http.ListenAndServe(":8001", r))
	}()
	if err := serverInstance.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
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
	err := db.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS post_stats (
            post_id UInt64,
            viewed UInt64,
            liked UInt64,
            timestamp DateTime
        ) engine = MergeTree()
        ORDER BY post_id
        PRIMARY KEY post_id;
    `)
	better_errors.CheckErrorFatal(err, "error on creating post_stats")
	err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS post_author (
			post_id UInt64,
			author_login String
		) engine = MergeTree()
		ORDER BY post_id
		PRIMARY KEY post_id;
    `)
	better_errors.CheckErrorFatal(err, "error on creating post_author")
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(context.Background()); err != nil {
		http.Error(w, "Clickhouse ping error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("All good.\n"))
}

func GetPostStats(postId uint64) (*pb.TGetPostStatsResponse, error) {
	row := db.QueryRow(context.Background(), fmt.Sprintf(`
		SELECT
			first_value(post_id) as PostId,
			sum(viewed) as Views,
			sum(liked) as Likes
		FROM post_stats
		WHERE post_id == %v
	`, postId))
	stats := &pb.TGetPostStatsResponse{}
	err := row.ScanStruct(stats)
	return stats, err
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

	pbRes, err := GetPostStats(postId)
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

func (s *server) GetTopPosts(ctx context.Context, request *pb.TGetTopPostsRequest) (*pb.TGetTopPostsResponse, error) {
	query := fmt.Sprintf(`
	SELECT
		ps.post_id,
		pa.author_login,
		SUM(ps.liked) AS total_likes,
		SUM(ps.viewed) AS total_views
	FROM
	 	post_stats AS ps
	INNER JOIN
	 	post_author AS pa
	ON
	 	ps.post_id = pa.post_id
	GROUP BY
	 	ps.post_id, pa.author_login
	ORDER BY
	 	total_%v DESC
	LIMIT 3;
   `, request.OrderBy)

	// Execute the query
	rows, err := db.Query(ctx, query)
	if err != nil {
		log.Printf("failed to get top posts: %v", err)
		return nil, err
	}
	defer rows.Close()

	// Process the results
	response := &pb.TGetTopPostsResponse{}
	for rows.Next() {
		var postID uint64
		var totalLikes uint64
		var totalViews uint64
		var authorLogin string

		err := rows.Scan(&postID, &authorLogin, &totalLikes, &totalViews)
		if err != nil {
			log.Printf("error reading row: %v", err)
			return nil, err
		}

		response.Posts = append(response.Posts, &pb.TGetTopPostsResponse_TPostStat{
			PostId:      postID,
			AuthorLogin: authorLogin,
			Likes:       totalLikes,
			Views:       totalViews,
		})
	}

	// Check for any errors encountered during iteration
	if err = rows.Err(); err != nil {
		log.Printf("error iterating over rows: %v", err)
		return nil, err
	}

	// Return the response
	return response, nil

}

func (s *server) GetTopAuthors(ctx context.Context, _ *emptypb.Empty) (*pb.TGetTopAuthorsResponse, error) {
	rows, err := db.Query(context.Background(), `
		SELECT
			pa.author_login,
			COALESCE(SUM(ps.liked), 0) AS total_likes
		FROM
			post_stats AS ps
		INNER JOIN
			post_author AS pa
		ON
			ps.post_id = pa.post_id
		GROUP BY
			pa.author_login
		ORDER BY
			total_likes DESC
		LIMIT 3;
	`)
	if err != nil {
		log.Printf("failed to get top authors: %v", err)
		return nil, err
	}
	defer rows.Close()

	response := &pb.TGetTopAuthorsResponse{}
	for rows.Next() {
		var authorLogin string
		var totalLikes uint64

		err := rows.Scan(&authorLogin, &totalLikes)
		if err != nil {
			log.Printf("error reading row: %v", err)
			return nil, err
		}

		response.Authors = append(response.Authors, &pb.TGetTopAuthorsResponse_TAuthor{
			AuthorLogin: authorLogin,
			Likes:       totalLikes,
		})
	}

	if err = rows.Err(); err != nil {
		log.Printf("error iterating over rows: %v", err)
		return nil, err
	}

	return response, nil

}

func (s *server) GetPostStats(ctx context.Context, request *pb.TGetPostStatsRequest) (*pb.TGetPostStatsResponse, error) {
	stats, err := GetPostStats(request.GetPostId())
	return stats, err
}

func (s *server) AddPost(ctx context.Context, request *pb.TAddPostRequest) (*emptypb.Empty, error) {
	err := db.AsyncInsert(
		ctx,
		fmt.Sprintf(`INSERT INTO post_author (post_id, author_login) VALUES (%d, '%s')`, request.PostId, request.AuthorLogin),
		true,
	)
	return &emptypb.Empty{}, err
}
