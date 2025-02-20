package main

import (
	"context"
	"os"

	"github.com/Kryniol/go-moniepoint-storage-engine/application/cli"
	"github.com/Kryniol/go-moniepoint-storage-engine/application/container"
	"github.com/Kryniol/go-moniepoint-storage-engine/config"
)

func main() {
	keydirPath, ok := os.LookupEnv("DATA_PATH")
	if !ok {
		panic("Missing DATA_PATH env var")
	}

	cfg := config.Config{DataPath: keydirPath, Mode: config.Leader}
	ctx := context.Background()
	c, err := container.NewServiceContainer(ctx, cfg)
	if err != nil {
		panic(err)
	}

	app := cli.NewApp(c)
	app.Run(ctx)
}
