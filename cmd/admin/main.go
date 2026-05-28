package main

import (
	"log"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[admin] no .env file found, using environment variables")
	}
	Execute()
}
