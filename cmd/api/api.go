package main

import (
	"log"
	"net/http"
	"time"

	"github.com/annguyen0511/social/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type application struct {
	config config
	store  store.Storage
}

type config struct {
	addr     string
	dbConfig dbConfig
	env      string
}

type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

func (app *application) mount() *chi.Mux {
	r := chi.NewRouter()
	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr) // pick one ClientIPFrom* based on your infra, see below
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", app.healthCheckHandler)

		r.Route("/posts", func(r chi.Router) {
			r.Post("/", app.createPostHandler)
			r.Route("/{postID}", func(r chi.Router) {
				r.Use(app.postContextMiddileware)

				r.Get("/", app.getPostHandler)
				r.Post("/comment", app.createCommentHandler)
				r.Patch("/", app.updatePostHandler)
				r.Delete("/", app.deletePostHandler)
			})
		})
		r.Route("/users", func(r chi.Router) {
			r.Post("/register", app.registerHandler)
			// r.Post("/login", app.loginHandler)
			r.Route("/{userID}", func(r chi.Router) {
				r.Use(app.userContextMiddleware)

				r.Get("/", app.getUserHandler)
				// r.Patch("/",)
			})
			// r.Group(func(r chi.Router) {
			// 	r.Get("feed", app.createCommentHandler)
			// })
		})

		// The {userID} below is always the target of the action
		r.Route("/friend-ship", func(r chi.Router) {
			r.Get("/followers", app.listFollowersHandler)
			r.Get("/following", app.listFollowingHandler)
			r.Get("/blocking", app.listBlockingHandler)
			r.Get("/close-friends", app.listCloseFriendsHandler)

			r.Route("/{userID}", func(r chi.Router) {
				r.Use(app.userContextMiddleware)

				r.Get("/", app.friendshipStatusHandler)

				r.Put("/follow", app.followUserHandler)
				r.Delete("/follow", app.unfollowUserHandler)

				r.Put("/block", app.blockUserHandler)
				r.Delete("/block", app.unblockUserHandler)

				r.Put("/close-friend", app.addCloseFriendHandler)
				r.Delete("/close-friend", app.removeCloseFriendHandler)
			})
		})
	})
	return r
}

func (app *application) run(mux *chi.Mux) error {

	srv := http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	log.Println("starting server on :", app.config.addr)
	return srv.ListenAndServe()
}
