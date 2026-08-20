package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"kn-system/internal/auth"
	"kn-system/internal/config"
	"kn-system/internal/embedder"
	"kn-system/internal/handler"
	"kn-system/internal/llm"
	"kn-system/internal/middleware"
	"kn-system/internal/model"
	"kn-system/internal/parser"
	"kn-system/internal/queue"
	"kn-system/internal/repository"
	"kn-system/internal/service"
	"kn-system/internal/storage"
	"kn-system/internal/web"
	"kn-system/pkg/logger"

	"go.uber.org/zap"
)

// Server is the composition root: it wires concrete implementations of every
// interface together and owns the HTTP server lifecycle. Keeping this in one
// place makes the dependency graph legible and lets the individual packages stay
// single-purpose (each only knows the interfaces it depends on).
type Server struct {
	http   *http.Server
	queue  queue.Queue
	router *gin.Engine
}

// New constructs the whole dependency graph from a Config. Anything swappable
// (storage, embedder, llm, queue) is selected here from config so the same
// binary runs in dev (local+mock) or prod (minio+openai).
func New(cfg *config.Config) (*Server, error) {
	ctx := context.Background()

	// --- Infrastructure ---
	db, err := repository.NewDB(ctx, cfg.Database, "/app/migrations")
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}

	store, err := storage.New(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}

	emb, err := embedder.New(cfg.Embedder)
	if err != nil {
		return nil, fmt.Errorf("init embedder: %w", err)
	}
	ll, err := llm.New(cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("init llm: %w", err)
	}

	// --- Repositories (one per aggregate root) ---
	userRepo := repository.NewUserRepo(db)
	kbRepo := repository.NewKBRepo(db)
	docRepo := repository.NewDocumentRepo(db)
	qaRepo := repository.NewQARepo(db)

	// --- Parsers + chunker (stateless, reusable) ---
	registry := parser.NewRegistry(
		parser.PDFParser{},
		parser.DocxParser{},
		parser.PlainParser{},
	)
	chunker := parser.NewChunker(800, 120)

	// --- Services ---
	authSvc := service.NewAuthService(userRepo, auth.NewPassword(), auth.NewJWT(cfg.JWT))
	kbSvc := service.NewKBService(kbRepo)
	indexSvc := service.NewIndexingService(docRepo, store, registry, chunker, emb)
	docSvc := service.NewDocumentService(docRepo, kbRepo, store, nil, 50*1024*1024)
	qaSvc := service.NewQAService(docRepo, qaRepo, emb, ll, 5)
	dashSvc := service.NewDashboardService(qaRepo)

	// --- Queue (depends on the indexing processor) ---
	q, err := queue.New(cfg.Queue.Provider, indexSvc, cfg.Queue.Concurrency)
	if err != nil {
		return nil, fmt.Errorf("init queue: %w", err)
	}
	// Inject the queue into the document service now that it exists. Done via a
	// setter to avoid a constructor ordering cycle: the document service needs
	// the queue to enqueue, and the queue needs the indexing service, which is
	// independent. The setter keeps the dependency explicit.
	docSvc.SetQueue(q)

	// --- Handlers ---
	authH := handler.NewAuthHandler(authSvc)
	kbH := handler.NewKBHandler(kbSvc)
	docH := handler.NewDocumentHandler(docSvc)
	qaH := handler.NewQAHandler(qaSvc)
	dashH := handler.NewDashboardHandler(dashSvc)

	router := buildRouter(cfg, authSvc, authH, kbH, docH, qaH, dashH)

	// Serve the compiled SPA (if present) as the fallback for non-API routes.
	spa := web.New("/app/web")
	if spa.Enabled() {
		spa.Register(router)
		logger.L.Info("serving SPA from /app/web")
	} else {
		logger.L.Info("no SPA bundle found at /app/web; API-only mode")
	}

	srv := &Server{
		http: &http.Server{
			Addr:              ":" + cfg.Server.Port,
			Handler:           router,
			ReadHeaderTimeout: 10 * time.Second,
		},
		queue:  q,
		router: router,
	}
	return srv, nil
}

// Start runs the HTTP server and the background queue. The queue is started
// first so it is ready to receive jobs before any upload can arrive.
func (s *Server) Start(ctx context.Context) error {
	if err := s.queue.Start(ctx); err != nil {
		return fmt.Errorf("start queue: %w", err)
	}
	logger.L.Info("http server starting", zap.String("addr", s.http.Addr))
	if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown stops the HTTP server (rejecting new connections) and drains the
// queue. In-flight requests are given cfg.Server.ShutdownTimeout to finish,
// satisfying the graceful-shutdown non-functional requirement.
func (s *Server) Shutdown(ctx context.Context) error {
	logger.L.Info("graceful shutdown begin")
	if err := s.http.Shutdown(ctx); err != nil {
		logger.L.Error("http shutdown error", zap.Error(err))
	}
	if err := s.queue.Stop(); err != nil {
		logger.L.Error("queue stop error", zap.Error(err))
	}
	logger.L.Info("graceful shutdown complete")
	return nil
}

// buildRouter assembles the middleware chain and every route. Role gates are
// applied per route group so the authorization policy is visible in one place.
func buildRouter(
	cfg *config.Config,
	authSvc *service.AuthService,
	authH *handler.AuthHandler,
	kbH *handler.KBHandler,
	docH *handler.DocumentHandler,
	qaH *handler.QAHandler,
	dashH *handler.DashboardHandler,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.Recover())
	r.Use(middleware.RequestLog())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Health check is unauthenticated so docker/Compose can probe readiness.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")

	// --- Auth: public register/login, authed /me ---
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.GET("/me", middleware.Auth(authSvc), authH.Me)
	}

	// --- Everything below requires authentication ---
	secured := v1.Group("", middleware.Auth(authSvc))

	// Knowledge bases: any authenticated user can list/create; delete is admin.
	kb := secured.Group("/kb")
	{
		kb.GET("", kbH.List)
		kb.POST("", kbH.Create)
		kb.GET("/:id", kbH.Get)
		kb.PUT("/:id", kbH.Update)
		kb.DELETE("/:id", middleware.RequireRole(model.RoleAdmin), kbH.Delete)
		kb.POST("/:id/invite", kbH.Invite)
		// Documents nested under a kb for upload/list. Uses the same :id
		// param name as the other kb routes to avoid Gin's conflicting-param
		// panic (two different names at the same path position is not allowed).
		kb.POST("/:id/documents", docH.Upload)
		kb.GET("/:id/documents", docH.List)
	}

	// Documents addressed by id (detail, status, delete, download, reindex).
	docs := secured.Group("/documents")
	{
		docs.GET("/:id", docH.Get)
		docs.GET("/:id/status", docH.Status)
		docs.DELETE("/:id", docH.Delete)
		docs.GET("/:id/download", docH.Download)
		docs.POST("/:id/reindex", docH.Retry)
	}

	// QA + search + dashboard.
	qa := secured.Group("/qa")
	{
		qa.POST("/ask", qaH.Ask)
		qa.GET("/history", qaH.History)
		qa.POST("/feedback", qaH.Feedback)
	}
	secured.GET("/search", docH.Search)
	secured.GET("/dashboard",
		middleware.RequireRole(model.RoleAdmin, model.RoleManager), dashH.Load)

	return r
}
