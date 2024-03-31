package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	pb "proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	_ "github.com/lib/pq"
)

type server struct {
	pb.UnimplementedPostServiceServer
}

var db *sql.DB

func ConnectDB() (*sql.DB, error) {

	db, err := sql.Open("postgres", "host=postgres_post port=5432 user=post_service password=22848 dbname=posts_db sslmode=disable")

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS POSTS (
		PostId SERIAL PRIMARY KEY,
		Title TEXT,
		Content TEXT,
		AuthorLogin TEXT
	);
	`)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func (s *server) CreatePost(ctx context.Context, request *pb.TCreatePostRequest) (*pb.TCreatePostResponse, error) {
	var postId uint64
	err := db.QueryRowContext(
		ctx,
		"INSERT INTO POSTS (Title, Content, AuthorLogin) VALUES ($1, $2, $3) RETURNING PostId",
		request.Title,
		request.Content,
		request.AuthorLogin,
	).Scan(&postId)
	if err != nil {
		return &pb.TCreatePostResponse{Created: false}, status.Errorf(codes.Internal, "failed to create post: %v", err)
	}
	return &pb.TCreatePostResponse{Created: true, PostId: &postId}, nil
}

func (s *server) UpdatePost(ctx context.Context, req *pb.TUpdatePostRequest) (*pb.TUpdatePostResponse, error) {
	result, err := db.ExecContext(
		ctx,
		"UPDATE POSTS SET Title = $1, Content = $2 WHERE PostId = $3 and AuthorLogin = $4",
		req.Title,
		req.Content,
		req.PostId,
		req.AuthorLogin,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update post: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &pb.TUpdatePostResponse{Updated: false}, status.Errorf(codes.Internal, "failed to update post: %v", err)
	}
	if rowsAffected == 0 {
		return &pb.TUpdatePostResponse{Updated: false}, status.Errorf(codes.PermissionDenied, "Author change is restricted")
	}

	return &pb.TUpdatePostResponse{Updated: true}, nil
}

func (s *server) DeletePost(ctx context.Context, req *pb.TDeletePostRequest) (*pb.TDeletePostResponse, error) {
	result, err := db.ExecContext(ctx, "DELETE FROM POSTS WHERE PostId = $1 and AuthorLogin = $2", req.PostId, req.AuthorLogin)
	if err != nil {
		return &pb.TDeletePostResponse{Deleted: false}, status.Errorf(codes.Internal, "Failed to update post: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return &pb.TDeletePostResponse{Deleted: false}, status.Errorf(codes.Internal, "Failed to delete post: %v", err)
	}

	if rowsAffected == 0 {
		result, _ := db.ExecContext(ctx, "SELECT FROM POSTS WHERE PostId = $1", req.PostId)
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected != 0 {
			return &pb.TDeletePostResponse{Deleted: false}, status.Errorf(codes.PermissionDenied, "You are not the author of this post")
		}
		return &pb.TDeletePostResponse{Deleted: false}, status.Errorf(codes.NotFound, "Post not found")
	}

	return &pb.TDeletePostResponse{Deleted: true}, nil
}

func (s *server) GetPostById(ctx context.Context, req *pb.TGetPostByIdRequest) (*pb.TGetPostByIdResponse, error) {
	var post pb.TPost
	err := db.QueryRowContext(
		ctx,
		"SELECT PostId, Title, Content, AuthorLogin FROM POSTS WHERE PostId = $1",
		req.PostId,
	).Scan(&post.PostId, &post.Title, &post.Content, &post.AuthorLogin)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Post not found")
	}

	return &pb.TGetPostByIdResponse{Post: &post}, nil
}

func (s *server) GetPostsOnPage(ctx context.Context, req *pb.TGetPostsOnPageRequest) (*pb.TGetPostsOnPageResponse, error) {
	rows, err := db.QueryContext(ctx, "SELECT PostId, Title, Content, AuthorLogin FROM POSTS LIMIT $1 OFFSET $2", 10, req.PageId*10)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch posts: %v", err)
	}
	defer rows.Close()

	var posts []*pb.TPost
	for rows.Next() {
		var post pb.TPost
		if err := rows.Scan(&post.PostId, &post.Title, &post.Content, &post.AuthorLogin); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan post: %v", err)
		}
		posts = append(posts, &post)
	}

	if err := rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "error iterating over rows: %v", err)
	}

	return &pb.TGetPostsOnPageResponse{
		Posts: posts,
	}, nil
}

func main() {
	listenAddress := ":50051"
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Printf("Failed to listen: %v", err)
		os.Exit(1)
	}

	db, err = ConnectDB()
	if err != nil {
		log.Printf("failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	serverInstance := grpc.NewServer()
	pb.RegisterPostServiceServer(serverInstance, &server{})

	fmt.Printf("Server is running at %v.\n", listenAddress)

	if err := serverInstance.Serve(lis); err != nil {
		log.Printf("Failed to serve: %v", err)
	}

}
