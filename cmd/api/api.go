package main

import (
	"net/http"
	"net/url"
	"time"

	"github.com/annguyen0511/social/docs" // This is required to generate swagger docs
	"github.com/annguyen0511/social/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
)

type application struct {
	config config
	store  store.Storage
	logger *zap.SugaredLogger
}

type config struct {
	addr     string
	dbConfig dbConfig
	env      string
	apiURL   string
	// invitationExp is how long a registration invitation stays valid.
	// invitationExp là thời gian một lời mời đăng ký còn hiệu lực.
	invitationExp time.Duration
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

		// Relative to /v1/swagger/index.html, so it resolves to
		// /v1/swagger/doc.json on whatever host and port serves the page.
		r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("doc.json")))

		r.Route("/authentication", func(r chi.Router) {
			r.Post("/register", app.registerHandler)
			r.Put("/activate/{token}", app.activateUserHandler)
			// r.Post("/login", app.loginHandler)
		})

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
			// r.Post("/login", app.loginHandler)
			r.Route("/{userID}", func(r chi.Router) {
				r.Use(app.userContextMiddleware)

				r.Get("/", app.getUserHandler)
				// r.Patch("/",)
			})
			r.Group(func(r chi.Router) {
				r.Get("/feed", app.getFeedHandler)
			})
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

	// This is required to set for swagger docs
	docs.SwaggerInfo.Version = version
	docs.SwaggerInfo.BasePath = "/v1"

	// Swagger 2.0 wants host without the scheme ("localhost:8080"), and the
	// scheme listed separately. Passing "http://localhost:8080" as the host
	// makes "Try it out" call http://http://localhost:8080/v1/...
	if u, err := url.Parse(app.config.apiURL); err == nil && u.Host != "" {
		docs.SwaggerInfo.Host = u.Host
		docs.SwaggerInfo.Schemes = []string{u.Scheme}
	}

	srv := http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	app.logger.Infow("starting server on", "address", app.config.addr, "env", app.config.env)
	return srv.ListenAndServe()
}
