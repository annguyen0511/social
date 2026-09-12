package main

import (
	"context"
	"flag"
	"log"

	"github.com/annguyen0511/social/internal/db"
	"github.com/annguyen0511/social/internal/env"
	"github.com/annguyen0511/social/internal/store"
)

func main() {
	numUsers := flag.Int("users", 100, "number of users to generate")
	numPosts := flag.Int("posts", 200, "number of posts to generate")
	numComments := flag.Int("comments", 500, "number of comments to generate")
	reset := flag.Bool("reset", false, "truncate users, posts and comments before seeding")
	flag.Parse()

	addr := env.GetString("DB_ADDR", "postgres://localhost:5432/socialnetwork?sslmode=disable")

	conn, err := db.New(addr, 10, 10, "15m")
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}
	defer conn.Close()

	ctx := context.Background()

	if *reset {
		if err := db.Reset(ctx, conn); err != nil {
			log.Fatal("failed to reset database: ", err)
		}
	}

	if err := db.Seed(ctx, store.NewStorage(conn), *numUsers, *numPosts, *numComments); err != nil {
		log.Fatal("failed to seed database: ", err)
	}

	log.Println("seeding complete")
}
