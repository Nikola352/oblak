package httpserver

import (
	"oblak/internal/function"
	"oblak/internal/invocation"
	"oblak/internal/server/events"
	"oblak/internal/server/ratelimiter"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"

	"oblak/internal/server/authkey"
	"oblak/internal/server/handler"
	"oblak/internal/server/middleware"
)

type Server struct {
	router    *gin.Engine
	db        *pgxpool.Pool
	filestore *minio.Client
}

func New(db *pgxpool.Pool, filestore *minio.Client, kek string, quarantineBus *events.Bus[events.QuarantineEvent],
	extractionBus *events.Bus[events.ExtractionEvent], functionStore *function.Store, invocationStore *invocation.Store,
	tokenBucket *ratelimiter.TokenBucket) *Server {
	s := &Server{
		router:    gin.Default(),
		db:        db,
		filestore: filestore,
	}
	h := handler.New(functionStore, invocationStore, filestore, quarantineBus, extractionBus, tokenBucket)
	auth := middleware.RequireAuth(authkey.NewStore(db, kek))
	s.registerRoutes(h, auth)
	return s
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
