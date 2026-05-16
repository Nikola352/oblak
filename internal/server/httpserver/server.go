package httpserver

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"

	"oblak/internal/server/handler"
)

type Server struct {
	router    *gin.Engine
	db        *pgxpool.Pool
	filestore *minio.Client
}

func New(db *pgxpool.Pool, filestore *minio.Client) *Server {
	s := &Server{
		router:    gin.Default(),
		db:        db,
		filestore: filestore,
	}
	h := handler.New()
	s.registerRoutes(h, nil)
	return s
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
