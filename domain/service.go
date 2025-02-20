package domain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type WriteStorage interface {
	Set(ctx context.Context, key string, data []byte) error
	Delete(ctx context.Context, key string) error
}

type ReadStorage interface {
	Get(ctx context.Context, key string) ([]byte, error)
	GetRange(ctx context.Context, startKey, endKey string) (map[string][]byte, error)
	GetSince(ctx context.Context, since time.Time) (map[string][]byte, error)
	GetAll(ctx context.Context) (map[string][]byte, error)
}

type KeyIndex interface {
	Find(ctx context.Context, key string) (*IndexEntry, error)
	Add(ctx context.Context, entry IndexEntry) error
}

type RangeIndex interface {
	Add(ctx context.Context, entry IndexEntry) error
	FindRange(ctx context.Context, start, end IndexEntry) (map[string]IndexEntry, error)
	FindSince(ctx context.Context, since time.Time) (map[string]IndexEntry, error)
	FindFirst(ctx context.Context) (*IndexEntry, error)
	FindLast(ctx context.Context) (*IndexEntry, error)
}

type Index interface {
	Find(ctx context.Context, key string) (*IndexEntry, error)
	Add(ctx context.Context, entry IndexEntry) error
	FindRange(ctx context.Context, startKey, endKey string) (map[string]IndexEntry, error)
	FindSince(ctx context.Context, since time.Time) (map[string]IndexEntry, error)
	FindFirst(ctx context.Context) (*IndexEntry, error)
	FindLast(ctx context.Context) (*IndexEntry, error)
}

type KeydirIndexer interface {
	Reindex(ctx context.Context) ([]IndexEntry, error)
}

type StorageEngine interface {
	Put(ctx context.Context, key string, data []byte) error
	Read(ctx context.Context, key string) ([]byte, error)
	ReadKeyRange(ctx context.Context, startKey, endKey string) (map[string][]byte, error)
	ReadSince(ctx context.Context, since *time.Time) (map[string][]byte, error)
	BatchPut(ctx context.Context, data map[string][]byte) error
	Delete(ctx context.Context, key string) error
}

type ReplicaSync interface {
	SyncFromLeader(ctx context.Context) error
}

type LeaderDataFetcher interface {
	FetchSince(ctx context.Context, since time.Time) (map[string][]byte, error)
	FetchAll(ctx context.Context) (map[string][]byte, error)
}

type storageEngine struct {
	r ReadStorage
	w WriteStorage
}

func NewStorageEngine(r ReadStorage, w WriteStorage) *storageEngine {
	return &storageEngine{r: r, w: w}
}

func (e *storageEngine) Put(ctx context.Context, key string, data []byte) error {
	err := e.w.Set(ctx, key, data)

	if err != nil {
		return fmt.Errorf("failed to update data in write storage: %w", err)
	}

	return nil
}

func (e *storageEngine) Read(ctx context.Context, key string) ([]byte, error) {
	v, err := e.r.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from read storage: %w", err)
	}

	return v, nil
}

func (e *storageEngine) ReadKeyRange(ctx context.Context, startKey, endKey string) (map[string][]byte, error) {
	vals, err := e.r.GetRange(ctx, startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from read storage: %w", err)
	}

	return vals, nil
}

func (e *storageEngine) ReadSince(ctx context.Context, since *time.Time) (map[string][]byte, error) {
	if since == nil {
		return e.r.GetAll(ctx)
	}

	vals, err := e.r.GetSince(ctx, *since)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from read storage: %w", err)
	}

	return vals, nil
}

func (e *storageEngine) BatchPut(ctx context.Context, data map[string][]byte) error {
	var wg sync.WaitGroup // could be replaced with errgroup if external packages were allowed
	errCh := make(chan error, len(data))
	for key, val := range data {
		wg.Add(1)
		go func(k string, v []byte) {
			defer wg.Done()
			if err := e.w.Set(ctx, k, v); err != nil {
				errCh <- fmt.Errorf("failed to update data in write storage for key %s: %w", k, err)
			}
		}(key, val)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to batch update data: %w", errors.Join(errs...))
	}

	return nil
}

func (e *storageEngine) Delete(ctx context.Context, key string) error {
	err := e.w.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete data from write storage: %w", err)
	}

	return nil
}

type replicaSync struct {
	rangeIndex        RangeIndex
	leaderDataFetcher LeaderDataFetcher
	writeStorage      WriteStorage
}

func NewReplicaSync(
	rangeIndex RangeIndex,
	leaderDataFetcher LeaderDataFetcher,
	writeStorage WriteStorage,
) *replicaSync {
	return &replicaSync{rangeIndex: rangeIndex, leaderDataFetcher: leaderDataFetcher, writeStorage: writeStorage}
}

func (e *replicaSync) SyncFromLeader(ctx context.Context) error {
	last, err := e.rangeIndex.FindLast(ctx)
	if err != nil {
		return fmt.Errorf("failed to get last index entry: %w", err)
	}

	var data map[string][]byte
	if last != nil {
		data, err = e.leaderDataFetcher.FetchSince(ctx, last.Timestamp)
		if err != nil {
			return fmt.Errorf("failed to fetch latest data from leader: %w", err)
		}
	} else {
		data, err = e.leaderDataFetcher.FetchAll(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch all data from leader: %w", err)
		}
	}

	for key, val := range data {
		err = e.writeStorage.Set(ctx, key, val)
		if err != nil {
			return fmt.Errorf("failed to store new data from leader for key %s: %w", key, err)
		}
	}

	return nil
}

type indexFacade struct {
	keyIndex   KeyIndex
	rangeIndex RangeIndex
}

func NewIndexFacade(keyIndex KeyIndex, rangeIndex RangeIndex) *indexFacade {
	return &indexFacade{keyIndex: keyIndex, rangeIndex: rangeIndex}
}

func (c *indexFacade) Find(ctx context.Context, key string) (*IndexEntry, error) {
	return c.keyIndex.Find(ctx, key)
}

func (c *indexFacade) Add(ctx context.Context, entry IndexEntry) error {
	err := c.keyIndex.Add(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to add entry to key index: %w", err)
	}

	err = c.rangeIndex.Add(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to add entry to range index: %w", err)
	}

	return nil
}

func (c *indexFacade) FindRange(ctx context.Context, startKey, endKey string) (map[string]IndexEntry, error) {
	start, err := c.keyIndex.Find(ctx, startKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find start key %s in key index: %w", startKey, err)
	}

	end, err := c.keyIndex.Find(ctx, endKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find end key %s in key index: %w", endKey, err)
	}

	return c.rangeIndex.FindRange(ctx, *start, *end)
}

func (c *indexFacade) FindSince(ctx context.Context, since time.Time) (map[string]IndexEntry, error) {
	return c.rangeIndex.FindSince(ctx, since)
}

func (c *indexFacade) FindFirst(ctx context.Context) (*IndexEntry, error) {
	return c.rangeIndex.FindFirst(ctx)
}

func (c *indexFacade) FindLast(ctx context.Context) (*IndexEntry, error) {
	return c.rangeIndex.FindLast(ctx)
}

type compositeWriteStorage struct {
	storages []WriteStorage
}

func NewCompositeWriteStorage(storages ...WriteStorage) *compositeWriteStorage {
	return &compositeWriteStorage{storages: storages}
}

func (s *compositeWriteStorage) Set(ctx context.Context, key string, data []byte) error {
	for i, storage := range s.storages {
		err := storage.Set(ctx, key, data)
		if err != nil {
			return fmt.Errorf("failed to set key in %d storage: %w", i, err)
		}
	}

	return nil
}

func (s *compositeWriteStorage) Delete(ctx context.Context, key string) error {
	for i, storage := range s.storages {
		err := storage.Delete(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to delete key in %d storage: %w", i, err)
		}
	}

	return nil
}
