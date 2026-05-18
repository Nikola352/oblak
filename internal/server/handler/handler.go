package handler

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

type Handler struct {
	db        *pgxpool.Pool
	filestore *minio.Client
}

func New(db *pgxpool.Pool, filestore *minio.Client) *Handler {
	return &Handler{db: db, filestore: filestore}
}
