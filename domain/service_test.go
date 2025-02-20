package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kryniol/go-moniepoint-storage-engine/domain"
	domainmocks "github.com/Kryniol/go-moniepoint-storage-engine/mocks/domain"
)

func TestStorageEngine_ReadSince_WithTimestamp_ReturnsNarrowedEntries(t *testing.T) {
	ctx := context.Background()
	since := time.Now().Add(-time.Hour)
	expectedData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	mockReadStorage := domainmocks.NewMockReadStorage(t)
	mockReadStorage.On("GetSince", ctx, since).Return(expectedData, nil)

	engine := domain.NewStorageEngine(mockReadStorage, domainmocks.NewMockWriteStorage(t))
	result, err := engine.ReadSince(ctx, &since)

	require.NoError(t, err)
	assert.Equal(t, expectedData, result)
}

func TestStorageEngine_ReadSince_NoTimestamp_ReturnsAllEntries(t *testing.T) {
	ctx := context.Background()
	expectedData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	mockReadStorage := domainmocks.NewMockReadStorage(t)
	mockReadStorage.On("GetAll", ctx).Return(expectedData, nil)

	engine := domain.NewStorageEngine(mockReadStorage, domainmocks.NewMockWriteStorage(t))
	result, err := engine.ReadSince(ctx, nil)

	require.NoError(t, err)
	assert.Equal(t, expectedData, result)
}

func TestStorageEngine_BatchPut_Successful(t *testing.T) {
	ctx := context.Background()
	data := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	mockWriteStorage := domainmocks.NewMockWriteStorage(t)
	for key, value := range data {
		mockWriteStorage.On("Set", ctx, key, value).Return(nil)
	}

	engine := domain.NewStorageEngine(domainmocks.NewMockReadStorage(t), mockWriteStorage)
	err := engine.BatchPut(ctx, data)

	require.NoError(t, err)
}

func TestReplicaSync_SyncFromLeader_NoExistingEntries_FetchAll(t *testing.T) {
	ctx := context.Background()
	mockData := map[string][]byte{"key1": []byte("value1"), "key2": []byte("value2")}

	mockRangeIndex := domainmocks.NewMockRangeIndex(t)
	mockRangeIndex.On("FindLast", ctx).Return(nil, nil)
	mockLeaderDataFetcher := domainmocks.NewMockLeaderDataFetcher(t)
	mockLeaderDataFetcher.On("FetchAll", ctx).Return(mockData, nil)
	mockWriteStorage := domainmocks.NewMockWriteStorage(t)
	for key, val := range mockData {
		mockWriteStorage.On("Set", ctx, key, val).Return(nil)
	}

	sync := domain.NewReplicaSync(mockRangeIndex, mockLeaderDataFetcher, mockWriteStorage)
	err := sync.SyncFromLeader(ctx)

	require.NoError(t, err)
}

func TestReplicaSync_SyncFromLeader_WithExistingEntries_FetchSince(t *testing.T) {
	ctx := context.Background()
	lastEntry := &domain.IndexEntry{Key: "lastKey", Timestamp: time.Unix(100, 0)}
	mockData := map[string][]byte{"key3": []byte("value3")}

	mockRangeIndex := domainmocks.NewMockRangeIndex(t)
	mockRangeIndex.On("FindLast", ctx).Return(lastEntry, nil)
	mockLeaderDataFetcher := domainmocks.NewMockLeaderDataFetcher(t)
	mockLeaderDataFetcher.On("FetchSince", ctx, lastEntry.Timestamp).Return(mockData, nil)
	mockWriteStorage := domainmocks.NewMockWriteStorage(t)
	for key, val := range mockData {
		mockWriteStorage.On("Set", ctx, key, val).Return(nil)
	}

	sync := domain.NewReplicaSync(mockRangeIndex, mockLeaderDataFetcher, mockWriteStorage)
	err := sync.SyncFromLeader(ctx)

	require.NoError(t, err)
}

func TestIndexFacade_Find_Successful(t *testing.T) {
	ctx := context.Background()
	entry := &domain.IndexEntry{Key: "key1", Timestamp: time.Now()}

	mockKeyIndex := domainmocks.NewMockKeyIndex(t)
	mockKeyIndex.On("Find", ctx, "key1").Return(entry, nil)

	facade := domain.NewIndexFacade(mockKeyIndex, domainmocks.NewMockRangeIndex(t))
	result, err := facade.Find(ctx, "key1")

	require.NoError(t, err)
	assert.Equal(t, entry, result)
}

func TestIndexFacade_Add_Successful(t *testing.T) {
	ctx := context.Background()
	entry := domain.IndexEntry{Key: "key1", Timestamp: time.Now()}

	mockKeyIndex := domainmocks.NewMockKeyIndex(t)
	mockKeyIndex.On("Add", ctx, entry).Return(nil)
	mockRangeIndex := domainmocks.NewMockRangeIndex(t)
	mockRangeIndex.On("Add", ctx, entry).Return(nil)

	facade := domain.NewIndexFacade(mockKeyIndex, mockRangeIndex)
	err := facade.Add(ctx, entry)

	require.NoError(t, err)
}

func TestIndexFacade_FindRange_Successful(t *testing.T) {
	ctx := context.Background()
	startEntry := &domain.IndexEntry{Key: "start", Timestamp: time.Unix(1, 0)}
	endEntry := &domain.IndexEntry{Key: "end", Timestamp: time.Unix(10, 0)}
	expectedResult := map[string]domain.IndexEntry{
		"start": *startEntry,
		"mid":   {Key: "mid", Timestamp: time.Unix(5, 0)},
		"end":   *endEntry,
	}

	mockKeyIndex := domainmocks.NewMockKeyIndex(t)
	mockKeyIndex.On("Find", ctx, "start").Return(startEntry, nil)
	mockKeyIndex.On("Find", ctx, "end").Return(endEntry, nil)
	mockRangeIndex := domainmocks.NewMockRangeIndex(t)
	mockRangeIndex.On("FindRange", ctx, *startEntry, *endEntry).Return(expectedResult, nil)

	facade := domain.NewIndexFacade(mockKeyIndex, mockRangeIndex)
	result, err := facade.FindRange(ctx, "start", "end")

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

func TestIndexFacade_FindSince_Successful(t *testing.T) {
	ctx := context.Background()
	since := time.Now().Add(-time.Hour)
	expectedResult := map[string]domain.IndexEntry{
		"key1": {Key: "key1", Timestamp: since.Add(time.Minute)},
		"key2": {Key: "key2", Timestamp: since.Add(2 * time.Minute)},
	}

	mockRangeIndex := domainmocks.NewMockRangeIndex(t)
	mockRangeIndex.On("FindSince", ctx, since).Return(expectedResult, nil)

	facade := domain.NewIndexFacade(domainmocks.NewMockKeyIndex(t), mockRangeIndex)
	result, err := facade.FindSince(ctx, since)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

func TestIndexFacade_FindFirst_Successful(t *testing.T) {
	ctx := context.Background()
	firstEntry := &domain.IndexEntry{Key: "first", Timestamp: time.Unix(1, 0)}

	mockRangeIndex := domainmocks.NewMockRangeIndex(t)
	mockRangeIndex.On("FindFirst", ctx).Return(firstEntry, nil)

	facade := domain.NewIndexFacade(domainmocks.NewMockKeyIndex(t), mockRangeIndex)
	result, err := facade.FindFirst(ctx)

	require.NoError(t, err)
	assert.Equal(t, firstEntry, result)
}

func TestIndexFacade_FindLast_Successful(t *testing.T) {
	ctx := context.Background()
	lastEntry := &domain.IndexEntry{Key: "last", Timestamp: time.Unix(100, 0)}

	mockRangeIndex := domainmocks.NewMockRangeIndex(t)
	mockRangeIndex.On("FindLast", ctx).Return(lastEntry, nil)

	facade := domain.NewIndexFacade(domainmocks.NewMockKeyIndex(t), mockRangeIndex)
	result, err := facade.FindLast(ctx)

	require.NoError(t, err)
	assert.Equal(t, lastEntry, result)
}
