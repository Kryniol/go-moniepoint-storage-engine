package rest

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/Kryniol/go-moniepoint-storage-engine/application/container"
	"github.com/Kryniol/go-moniepoint-storage-engine/config"
)

type app struct {
	c   *container.Container
	cfg config.Config
	wg  *sync.WaitGroup
}

func NewApp(c *container.Container, cfg config.Config, wg *sync.WaitGroup) *app {
	return &app{c: c, cfg: cfg, wg: wg}
}

func (a *app) Run() func(ctx context.Context) error {
	if a.cfg.Mode == config.Follower {
		err := a.c.ReplicaSync.SyncFromLeader(context.Background())
		if err != nil {
			log.Fatalf("Failed to sync data from leader: %v", err)
		}

		log.Println("Data has been synchronized with the leader host")
	}

	h := NewStorageHandler(a.c)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", a.cfg.HTTPPort),
		Handler: LoggingMiddleware(http.DefaultServeMux),
	}

	http.HandleFunc(
		"/health-check", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		},
	)
	http.HandleFunc("/put", h.Put)
	http.HandleFunc("/read", h.Read)
	http.HandleFunc("/readrange", h.ReadKeyRange)
	http.HandleFunc("/batchput", h.BatchPut)
	http.HandleFunc("/delete", h.Delete)

	if a.cfg.Mode == config.Leader {
		http.HandleFunc("/replicate/readsince", h.ReplicateReadSince)
		http.HandleFunc("/replicate/readall", h.ReplicateReadAll)
	} else {
		http.HandleFunc("/replicate/put", h.ReplicatePut)
		http.HandleFunc("/replicate/delete", h.ReplicateDelete)
	}

	a.wg.Add(1)

	go func() {
		defer a.wg.Done()

		log.Printf("Starting REST server in %s mode at :%s", a.cfg.Mode, a.cfg.HTTPPort)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe() has crashed: %v", err)
		}
	}()

	return func(ctx context.Context) error {
		log.Println("Shutting REST server down gracefully")
		return srv.Shutdown(ctx)
	}
}
