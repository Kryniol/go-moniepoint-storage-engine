package container

import (
	"context"
	"fmt"
	"time"

	"github.com/Kryniol/go-moniepoint-storage-engine/config"
	"github.com/Kryniol/go-moniepoint-storage-engine/domain"
	"github.com/Kryniol/go-moniepoint-storage-engine/infrastructure"
)

type Container struct {
	Engine        domain.StorageEngine
	ReplicaEngine domain.StorageEngine
	ReplicaSync   domain.ReplicaSync
}

func NewServiceContainer(ctx context.Context, cfg config.Config) (*Container, error) {
	indexer := infrastructure.NewFilesystemKeydirIndexer(cfg.DataPath)
	entries, err := indexer.Reindex(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to reindex: %w", err)
	}

	keyIndex := domain.NewInMemoryKeyIndex(entries)
	rangeIndex, err := domain.NewSkipListRangeIndex(ctx, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize range index: %w", err)
	}

	index := domain.NewIndexFacade(keyIndex, rangeIndex)
	filesystemWriteStorage, err := infrastructure.NewFilesystemWriteStorage(index, cfg.DataPath, time.Now)
	filesystemReadStorage := infrastructure.NewFilesystemReadStorage(index, cfg.DataPath)
	if cfg.Mode == config.Leader {
		replicaFanoutWriteStorage := infrastructure.NewRESTReplicaFanoutWriteStorage(cfg.ReplicaHosts)
		compositeWriteStorage := domain.NewCompositeWriteStorage(filesystemWriteStorage, replicaFanoutWriteStorage)

		return &Container{Engine: domain.NewStorageEngine(filesystemReadStorage, compositeWriteStorage)}, nil
	}

	leaderDataFetcher := infrastructure.NewLeaderRESTDataFetcher(cfg.LeaderHost)
	replicaSync := domain.NewReplicaSync(rangeIndex, leaderDataFetcher, filesystemWriteStorage)
	restWriteStorage := infrastructure.NewRESTWriteStorage(cfg.LeaderHost)
	restWriteEngine := domain.NewStorageEngine(filesystemReadStorage, restWriteStorage)

	return &Container{
		Engine:        restWriteEngine,
		ReplicaEngine: domain.NewStorageEngine(filesystemReadStorage, filesystemWriteStorage),
		ReplicaSync:   replicaSync,
	}, nil
}
