package httpserver

import (
	"errors"
	"net/http"
	"oblak/internal/server/apperr"

	"github.com/gin-gonic/gin"

	"oblak/internal/api"
	"oblak/internal/server/handler"
)

func (s *Server) registerRoutes(h *handler.Handler, middlewares ...gin.HandlerFunc) {
	s.router.Use(middlewares...)
	api.RegisterHandlers(s.router, api.NewStrictHandlerWithOptions(h, nil, api.StrictGinServerOptions{
		ResponseErrorHandlerFunc: func(ctx *gin.Context, err error) {
			var appErr *apperr.Error
			if errors.As(err, &appErr) {
				ctx.JSON(appErr.Status, gin.H{"error": appErr.Message})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		},
	}))
}
