package server

import "github.com/gin-gonic/gin"

func (s *Server) registerRoutes() {
	s.router.GET("/health", s.healthCheck)
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
