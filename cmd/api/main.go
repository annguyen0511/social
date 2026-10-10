package main

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/annguyen0511/social/internal/auth"
	"github.com/annguyen0511/social/internal/db"
	"github.com/annguyen0511/social/internal/env"
	"github.com/annguyen0511/social/internal/mailer"
	"github.com/annguyen0511/social/internal/store"
	"github.com/annguyen0511/social/internal/upload"
	"go.uber.org/zap"
)

const version = "0.0.1"

//	@title			Social Network API
//	@version		0.0.1
//	@description	API for building social network.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						Authorization
// @description				Enter your JWT token here to access protected, Example: "Bearer {token}"
func main() {
	cfg := config{
		addr:   env.GetString("ADDR", ":8080"),
		apiURL: env.GetString("EXTERNAL_URL", "http://localhost:8080"),
		dbConfig: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://localhost:5432/socialnetwork?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 10),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
		env:         env.GetString("ENV", "development"),
		uploadDir:   env.GetString("UPLOAD_DIR", "./uploads"),
		corsOrigins: strings.Split(env.GetString("CORS_ALLOWED_ORIGINS", "http://localhost:5173"), ","),
		auth: authConfig{
			secret: env.GetString("JWT_SECRET", ""),
			issuer: env.GetString("JWT_ISSUER", "social-network"),
			exp:    env.GetDuration("JWT_EXP", 3*24*time.Hour),
		},
		mail: mailConfig{
			exp:         env.GetDuration("INVITATION_EXP", 3*24*time.Hour),
			fromEmail:   env.GetString("MAIL_FROM_EMAIL", "no-reply@example.com"),
			fromName:    env.GetString("MAIL_FROM_NAME", "Social Network"),
			frontendURL: env.GetString("FRONTEND_URL", "http://localhost:5173"),
			sendGrid: sendGridConfig{
				apiKey: env.GetString("SENDGRID_API_KEY", ""),
				// Sandbox lets SendGrid validate the request and answer 200
				// without delivering, so development never mails real people.
				//
				// Sandbox để SendGrid kiểm tra request rồi trả 200 mà không
				// gửi đi, nên lúc dev không bao giờ làm phiền người thật.
				sandbox: env.GetBool("MAIL_SANDBOX", true),
			},
		},
	}

	//logger
	logger := zap.Must(zap.NewProduction()).Sugar()

	defer logger.Sync()

	// database
	db, err := db.New(cfg.dbConfig.addr, cfg.dbConfig.maxOpenConns, cfg.dbConfig.maxIdleConns, cfg.dbConfig.maxIdleTime)

	if err != nil {
		logger.Panicw("failed to connect database", "error", err)
	}

	defer db.Close()

	// authenticator
	if cfg.auth.secret == "" {
		if cfg.env == "production" {
			logger.Fatal("JWT_SECRET is required when ENV=production")
		}
		// A fixed development secret keeps sessions alive across restarts.
		// It is deliberately obvious so it can never be mistaken for a real
		// one, and production refuses to start without a proper value.
		//
		// Một secret cố định cho dev để phiên đăng nhập không mất sau mỗi lần
		// khởi động lại. Nó cố tình lộ liễu để không bao giờ bị nhầm là
		// secret thật, và production thì từ chối chạy nếu thiếu giá trị thật.
		cfg.auth.secret = "insecure-development-secret-do-not-use-in-production"
		logger.Warnw("JWT_SECRET not set, using the insecure development secret")
	}
	authenticator := auth.NewJWT(cfg.auth.secret, cfg.auth.issuer, cfg.auth.exp)

	// mailer
	var mailClient mailer.Client
	switch {
	case cfg.mail.sendGrid.apiKey != "":
		mailClient = mailer.NewSendGrid(cfg.mail.sendGrid.apiKey, cfg.mail.fromEmail, cfg.mail.fromName, cfg.mail.sendGrid.sandbox)
		logger.Infow("mailer ready", "provider", "sendgrid", "sandbox", cfg.mail.sendGrid.sandbox)
	case cfg.env == "production":
		// Refusing to boot beats accepting registrations whose activation
		// mail silently goes nowhere.
		//
		// Thà không khởi động được còn hơn nhận đăng ký rồi thư kích hoạt
		// lặng lẽ không đi đâu cả.
		logger.Fatal("SENDGRID_API_KEY is required when ENV=production")
	default:
		mailClient = mailer.NewLog(logger.Infow)
		logger.Warnw("SENDGRID_API_KEY not set, activation links will only be logged")
	}

	// avatar storage
	// Two directories under the one upload root: avatars are public files
	// while post pictures are served through an authorised route, and
	// keeping them apart means a mistake in one cannot expose the other.
	//
	// Hai thư mục dưới cùng một gốc tải lên: avatar là file công khai còn ảnh
	// bài viết được phục vụ qua một route có kiểm quyền, tách riêng ra thì
	// một sai sót ở bên này không thể làm lộ bên kia.
	avatars, err := upload.NewStore(filepath.Join(cfg.uploadDir, "avatars"))
	if err != nil {
		logger.Panicw("failed to prepare upload directory", "dir", cfg.uploadDir, "error", err)
	}

	postImages, err := upload.NewStore(filepath.Join(cfg.uploadDir, "posts"))
	if err != nil {
		logger.Panicw("failed to prepare upload directory", "dir", cfg.uploadDir, "error", err)
	}

	logger.Infoln("database connection pool establised")
	store := store.NewStorage(db)

	app := &application{
		config:        cfg,
		store:         store,
		logger:        logger,
		mailer:        mailClient,
		authenticator: authenticator,
		avatars:       avatars,
		postImages:    postImages,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))
}
