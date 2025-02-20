package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/Kryniol/go-moniepoint-storage-engine/application/container"
	"github.com/Kryniol/go-moniepoint-storage-engine/application/rest"
	"github.com/Kryniol/go-moniepoint-storage-engine/config"
)

func main() {
	httpPort, ok := os.LookupEnv("HTTP_PORT")
	if !ok {
		panic("Missing HTTP_PORT env var")
	}

	dataPath, ok := os.LookupEnv("DATA_PATH")
	if !ok {
		panic("Missing DATA_PATH env var")
	}

	mode, ok := os.LookupEnv("REPLICATION_MODE")
	if !ok {
		panic("Missing REPLICATION_MODE env var")
	}

	leaderHost := os.Getenv("REPLICATION_LEADER_HOST")

	var replicaHosts []string
	replicaHostsStr, ok := os.LookupEnv("REPLICATION_REPLICA_HOSTS")
	if ok {
		replicaHosts = strings.Split(replicaHostsStr, ",")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.NewConfig(httpPort, dataPath, config.Mode(mode), leaderHost, replicaHosts)
	if err != nil {
		panic(err)
	}

	c, err := container.NewServiceContainer(ctx, *cfg)
	if err != nil {
		panic(err)
	}

	wg := &sync.WaitGroup{}
	app := rest.NewApp(c, *cfg, wg)
	shutdown := app.Run()

	go func() {
		<-ctx.Done()
		if err = shutdown(ctx); err != nil {
			panic(err)
		}
	}()

	wg.Wait()
}
