package main

import (
	"net/http"
	"net/url"
	"time"

	"github.com/annguyen0511/social/docs" // This is required to generate swagger docs
	"github.com/annguyen0511/social/internal/auth"
	"github.com/annguyen0511/social/internal/mailer"
	"github.com/annguyen0511/social/internal/store"
	"github.com/annguyen0511/social/internal/upload"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
)

type application struct {
	config        config
	store         store.Storage
	logger        *zap.SugaredLogger
	mailer        mailer.Client
	authenticator auth.Authenticator
	avatars       *upload.Store
}

type config struct {
	addr     string
	dbConfig dbConfig
	env      string
	apiURL   string
	mail     mailConfig
	auth     authConfig
	// corsOrigins lists the browser origins allowed to call the API. The
	// frontend runs on another port, so without this every request from it is
	// blocked before the handler ever runs.
	//
	// corsOrigins liệt kê các origin trình duyệt được phép gọi API. Frontend
	// chạy ở cổng khác, nên thiếu phần này thì mọi request từ nó bị chặn
	// trước cả khi handler chạy.
	corsOrigins []string
	// uploadDir is where avatar files are written. It sits outside the
	// repository tree by default, so a stray upload never ends up in a commit.
	//
	// uploadDir là nơi ghi các file avatar. Mặc định nó nằm ngoài cây mã nguồn,
	// để một file tải lên không bao giờ lọt vào commit.
	uploadDir string
}

type authConfig struct {
	secret string
	issuer string
	exp    time.Duration
}

type mailConfig struct {
	exp       time.Duration
	fromEmail string
	fromName  string
	sendGrid  sendGridConfig
	// frontendURL is where the activation link points. The API endpoint is a
	// PUT, which a mail client cannot follow, so the link goes to a page that
	// calls it.
	//
	// frontendURL là nơi đường dẫn kích hoạt trỏ tới. Endpoint của API là PUT
	// mà trình đọc mail không gọi được, nên link trỏ tới một trang rồi trang
	// đó gọi API.
	frontendURL string
}

type sendGridConfig struct {
	apiKey  string
	sandbox bool
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

	// AllowCredentials is what lets the browser send the session cookie to
	// another origin; it forbids the "*" wildcard, so the origins must be
	// listed explicitly.
	//
	// AllowCredentials là thứ cho phép trình duyệt gửi cookie phiên sang
	// origin khác; nó cấm dùng ký tự đại diện "*", nên phải liệt kê origin ra
	// cụ thể.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   app.config.corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", app.healthCheckHandler)

		// Relative to /v1/swagger/index.html, so it resolves to
		// /v1/swagger/doc.json on whatever host and port serves the page.
		r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("doc.json")))

		// Outside requireAuth so an <img> loads on its own. See
		// serveAvatarHandler.
		//
		// Nằm ngoài requireAuth để một thẻ <img> tự tải được. Xem
		// serveAvatarHandler.
		r.Get("/uploads/avatars/{name}", app.serveAvatarHandler)

		r.Route("/authentication", func(r chi.Router) {
			r.Post("/register", app.registerHandler)
			r.Put("/activate/{token}", app.activateUserHandler)
			r.Post("/login", app.loginHandler)
			r.Post("/logout", app.logoutHandler)
		})

		r.Route("/posts", func(r chi.Router) {
			r.Use(app.requireAuth)

			r.Post("/", app.createPostHandler)
			r.Route("/{postID}", func(r chi.Router) {
				r.Use(app.postContextMiddileware)

				r.Get("/", app.getPostHandler)
				r.Post("/comment", app.createCommentHandler)
				r.Put("/like", app.likeHandler)
				r.Delete("/like", app.unlikeHandler)
				r.Put("/save", app.saveHandler)
				r.Delete("/save", app.unsaveHandler)
				r.Put("/repost", app.repostHandler)
				r.Delete("/repost", app.unrepostHandler)
				r.Patch("/", app.updatePostHandler)
				r.Delete("/", app.deletePostHandler)
			})
		})
		r.Route("/users", func(r chi.Router) {
			r.Use(app.requireAuth)
			r.Route("/{userID}", func(r chi.Router) {
				r.Use(app.userContextMiddleware)

				r.Get("/", app.getUserHandler)
				r.Get("/posts", app.listUserPostsHandler)
				r.Get("/reposts", app.listUserRepostsHandler)
				r.Get("/followers", app.listFollowersOfUserHandler)
				r.Get("/following", app.listFollowingOfUserHandler)
			})
			r.Group(func(r chi.Router) {
				r.Get("/me", app.getCurrentUserHandler)
				r.Patch("/me", app.updateProfileHandler)
				r.Post("/me/avatar", app.uploadAvatarHandler)
				r.Delete("/me/avatar", app.deleteAvatarHandler)
				r.Get("/me/saved", app.listSavedHandler)
				r.Get("/feed", app.getFeedHandler)
				r.Get("/search", app.searchUsersHandler)
			})
		})

		// The {userID} below is always the target of the action
		r.Route("/friend-ship", func(r chi.Router) {
			r.Use(app.requireAuth)

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
