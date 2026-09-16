package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/relentlessworks/diffkit/internal/api"
	"github.com/relentlessworks/diffkit/internal/auth"
	"github.com/relentlessworks/diffkit/internal/config"
	"github.com/relentlessworks/diffkit/internal/store"
)

func main() {
	cfg := config.Load()

	s := store.New()
	h := api.New(s)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	var handler http.Handler = mux
	if cfg.APIKey != "" {
		handler = auth.APIKeyMiddleware(cfg.APIKey)(handler)
	}

	log.Printf("diffkit listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		fmt.Fprintf(log.Writer(), "error: %v\n", err)
	}
}
