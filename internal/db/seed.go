package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/annguyen0511/social/internal/model"
	"github.com/annguyen0511/social/internal/store"
)

// seedPassword is the plaintext password given to every generated user, so a
// seeded account can be logged into during local development. Hash this once
// registerHandler hashes passwords.
const seedPassword = "password123"

var firstNames = []string{
	"An", "Binh", "Chi", "Dung", "Giang", "Ha", "Hieu", "Khanh", "Lan", "Linh",
	"Mai", "Minh", "Nam", "Ngan", "Phuc", "Quang", "Son", "Thao", "Trang", "Tuan",
	"Alice", "Bob", "Carol", "Dave", "Erin", "Frank", "Grace", "Heidi", "Ivan", "Judy",
}

var lastNames = []string{
	"Nguyen", "Tran", "Le", "Pham", "Hoang", "Vu", "Dang", "Bui", "Do", "Ho",
	"Anderson", "Brown", "Clark", "Davis", "Evans", "Foster", "Green", "Hughes",
}

var postTitles = []string{
	"Why I stopped writing ORMs",
	"A tour of context cancellation in Go",
	"Postgres indexes you are probably missing",
	"Structuring a Go service without a framework",
	"Optimistic locking in practice",
	"Reading query plans without fear",
	"The case for boring technology",
	"chi vs net/http: what you actually gain",
	"Connection pools are not magic",
	"Migrations that survive a rollback",
	"Error wrapping conventions that scale",
	"Testing handlers without a database",
	"What sql.ErrNoRows really means",
	"Graceful shutdown, step by step",
	"Stop logging in your store layer",
}

var postContents = []string{
	"Spent the afternoon tracing a slow endpoint and the culprit was a missing index on a foreign key. Worth checking yours.",
	"Every query in this service now runs under a five second timeout. The failure mode changed from hanging to failing fast, which is a much better place to be.",
	"Optimistic locking is a version column and one extra predicate in the WHERE clause. That is the entire feature.",
	"Wrapping errors with %w costs nothing and lets the caller decide what matters. Do it from day one.",
	"I keep coming back to the standard library. Most of what a framework gives you is a router and some opinions.",
	"Connection pool sizing is not about throughput, it is about how much concurrency your database can actually absorb.",
	"Wrote the down migration first this time. It caught a column I would have been unable to drop cleanly.",
	"The handler should not know it is talking to Postgres. The store should not know it is answering an HTTP request.",
	"Reproduced the bug with a single curl once I stopped trusting what the client claimed it was sending.",
	"Seeding realistic data early makes pagination and sorting bugs show up long before production does.",
}

var postTags = []string{
	"go", "postgres", "sql", "api", "http", "chi", "testing", "performance",
	"migrations", "architecture", "backend", "debugging",
}

var commentContents = []string{
	"This matches what I ran into last week.",
	"Do you have benchmarks for that claim?",
	"Saved me a couple of hours, thanks.",
	"I would argue the opposite, but I see the reasoning.",
	"Any idea how this behaves under load?",
	"The second half is the part people skip.",
	"Bookmarking this one.",
	"Does this still hold with connection pooling in front?",
	"Clean write-up, easy to follow.",
	"We solved it differently but ended up in the same place.",
	"What happens when the context is cancelled mid-query?",
	"Small correction: that flag was renamed in the last release.",
}

// Seed fills the database with generated users, posts and comments. It is
// intended for local development only.
func Seed(ctx context.Context, s store.Storage, numUsers, numPosts, numComments int) error {
	users, err := seedUsers(ctx, s, numUsers)
	if err != nil {
		return err
	}
	log.Printf("seeded %d users", len(users))

	posts, err := seedPosts(ctx, s, users, numPosts)
	if err != nil {
		return err
	}
	log.Printf("seeded %d posts", len(posts))

	count, err := seedComments(ctx, s, users, posts, numComments)
	if err != nil {
		return err
	}
	log.Printf("seeded %d comments", count)

	return nil
}

// Reset removes every seeded row and restarts the identity sequences. Truncating
// users cascades into posts and comments.
func Reset(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `TRUNCATE comments, posts, users RESTART IDENTITY CASCADE`)
	if err != nil {
		return fmt.Errorf("reset tables: %w", err)
	}
	log.Println("truncated comments, posts and users")
	return nil
}

func seedUsers(ctx context.Context, s store.Storage, n int) ([]*model.User, error) {
	users := make([]*model.User, 0, n)
	for i := 0; i < n; i++ {
		first := pick(firstNames)
		last := pick(lastNames)
		// The suffix keeps username and email unique across repeated runs, so
		// seeding twice appends data instead of failing on the unique index.
		handle := fmt.Sprintf("%s.%s.%s", strings.ToLower(first), strings.ToLower(last), randomSuffix())

		user := &model.User{
			FirstName: first,
			LastName:  last,
			AvatarURL: fmt.Sprintf("https://i.pravatar.cc/150?u=%s", handle),
			UserName:  handle,
			Email:     handle + "@example.com",
			Password:  seedPassword,
		}

		if err := s.User.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("create user %q: %w", user.UserName, err)
		}
		users = append(users, user)
	}
	return users, nil
}

func seedPosts(ctx context.Context, s store.Storage, users []*model.User, n int) ([]*model.Post, error) {
	if len(users) == 0 {
		return nil, fmt.Errorf("cannot seed posts without users")
	}

	posts := make([]*model.Post, 0, n)
	for i := 0; i < n; i++ {
		post := &model.Post{
			Title:   pick(postTitles),
			Content: pick(postContents),
			UserID:  users[rand.Intn(len(users))].ID,
			Tags:    pickTags(1, 3),
		}

		if err := s.Post.Create(ctx, post); err != nil {
			return nil, fmt.Errorf("create post %q: %w", post.Title, err)
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func seedComments(ctx context.Context, s store.Storage, users []*model.User, posts []*model.Post, n int) (int, error) {
	if len(users) == 0 || len(posts) == 0 {
		return 0, fmt.Errorf("cannot seed comments without users and posts")
	}

	for i := 0; i < n; i++ {
		comment := &model.Comment{
			PostID:  posts[rand.Intn(len(posts))].ID,
			UserID:  users[rand.Intn(len(users))].ID,
			Content: pick(commentContents),
		}

		if err := s.Comment.Create(ctx, comment); err != nil {
			return i, fmt.Errorf("create comment: %w", err)
		}
	}
	return n, nil
}

func pick(pool []string) string {
	return pool[rand.Intn(len(pool))]
}

// pickTags returns between min and max distinct tags.
func pickTags(min, max int) []string {
	n := min + rand.Intn(max-min+1)
	chosen := make(map[string]struct{}, n)
	for len(chosen) < n {
		chosen[pick(postTags)] = struct{}{}
	}

	tags := make([]string, 0, n)
	for tag := range chosen {
		tags = append(tags, tag)
	}
	return tags
}

const suffixAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomSuffix() string {
	var b strings.Builder
	for i := 0; i < 5; i++ {
		b.WriteByte(suffixAlphabet[rand.Intn(len(suffixAlphabet))])
	}
	return b.String()
}
