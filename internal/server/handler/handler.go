package handler

import (
	"oblak/internal/server/function"

	"github.com/minio/minio-go/v7"
)

type Handler struct {
	functionStore *function.Store
	filestore     *minio.Client
}

func New(functionStore *function.Store, filestore *minio.Client) *Handler {
	return &Handler{functionStore: functionStore, filestore: filestore}
}
