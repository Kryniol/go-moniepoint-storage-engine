package domain

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryKeyIndex_Find_KeyExists_IndexIsReturned(t *testing.T) {
	entry := IndexEntry{
		Key:       "testKey",
		FileID:    "testFile",
		Timestamp: time.Now(),
	}

	index := NewInMemoryKeyIndex([]IndexEntry{entry})
	found, err := index.Find(context.Background(), "testKey")

	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, entry, *found)
}

func TestInMemoryKeyIndex_Find_KeyNotFound_CorrectErrorIsReturned(t *testing.T) {
	index := NewInMemoryKeyIndex(nil)
	found, err := index.Find(context.Background(), "missingKey")

	assert.Nil(t, found)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestInMemoryKeyIndex_AddAndFind_Successful(t *testing.T) {
	entry := IndexEntry{
		Key:       "newKey",
		FileID:    "newFile",
		Timestamp: time.Now(),
	}

	index := NewInMemoryKeyIndex(nil)
	err := index.Add(context.Background(), entry)
	require.NoError(t, err)

	found, err := index.Find(context.Background(), "newKey")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, entry, *found)
}

func TestInMemoryKeyIndex_ConcurrentAccess_Successful(t *testing.T) {
	var wg sync.WaitGroup
	index := NewInMemoryKeyIndex(nil)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			suffix := strconv.Itoa(i)
			entry := IndexEntry{
				Key:       "key" + suffix,
				FileID:    "file" + suffix,
				Timestamp: time.Now(),
			}
			err := index.Add(context.Background(), entry)
			assert.NoError(t, err)
		}(i)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = index.Find(context.Background(), "key"+strconv.Itoa(i))
		}(i)
	}

	wg.Wait()
}

func TestInMemoryKeyIndex_ConcurrentReadWrite_Successful(t *testing.T) {
	var wg sync.WaitGroup
	index := NewInMemoryKeyIndex(nil)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			suffix := strconv.Itoa(i)
			entry := IndexEntry{
				Key:       "concurrentKey" + suffix,
				FileID:    "file" + suffix,
				Timestamp: time.Now(),
			}
			_ = index.Add(context.Background(), entry)
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = index.Find(context.Background(), "concurrentKey"+strconv.Itoa(i))
		}(i)
	}

	wg.Wait()
}

func TestNewSkipListRangeIndex_InitialEntriesAreReturned(t *testing.T) {
	ctx := context.Background()
	entries := []IndexEntry{
		{Key: "key1", Timestamp: time.Unix(1, 0)},
		{Key: "key2", Timestamp: time.Unix(2, 0)},
	}

	index, err := NewSkipListRangeIndex(ctx, entries)
	require.NoError(t, err)
	require.NotNil(t, index)
	result, err := index.FindRange(ctx, entries[0], entries[1])

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, entries[0], result["key1"])
	assert.Equal(t, entries[1], result["key2"])
}

func TestSkipListRangeIndex_Add_Successful(t *testing.T) {
	ctx := context.Background()
	entry := IndexEntry{Key: "key1", Timestamp: time.Unix(10, 0)}

	index, _ := NewSkipListRangeIndex(ctx, nil)
	err := index.Add(ctx, entry)
	require.NoError(t, err)
	result, err := index.FindSince(ctx, time.Unix(0, 0))

	require.NoError(t, err)
	assert.Equal(t, entry, result["key1"])
}

func TestSkipListRangeIndex_FindRange_Successful(t *testing.T) {
	ctx := context.Background()
	entries := []IndexEntry{
		{Key: "key1", Timestamp: time.Unix(1, 0)},
		{Key: "key2", Timestamp: time.Unix(2, 0)},
		{Key: "key3", Timestamp: time.Unix(3, 0)},
	}

	index, _ := NewSkipListRangeIndex(ctx, entries)
	result, err := index.FindRange(ctx, entries[0], entries[2])

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, entries[0], result["key1"])
	assert.Equal(t, entries[1], result["key2"])
	assert.Equal(t, entries[2], result["key3"])
}

func TestSkipListRangeIndex_FindSince_Successful(t *testing.T) {
	ctx := context.Background()
	entries := []IndexEntry{
		{Key: "key1", Timestamp: time.Unix(1, 0)},
		{Key: "key2", Timestamp: time.Unix(5, 0)},
		{Key: "key3", Timestamp: time.Unix(10, 0)},
	}

	index, _ := NewSkipListRangeIndex(ctx, entries)
	result, err := index.FindSince(ctx, time.Unix(5, 0))

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, entries[1], result["key2"])
	assert.Equal(t, entries[2], result["key3"])
}

func TestSkipListRangeIndex_FindFirst_Successful(t *testing.T) {
	ctx := context.Background()
	entries := []IndexEntry{
		{Key: "key1", Timestamp: time.Unix(1, 0)},
		{Key: "key2", Timestamp: time.Unix(2, 0)},
	}

	index, _ := NewSkipListRangeIndex(ctx, entries)
	first, err := index.FindFirst(ctx)

	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, "key1", first.Key)
}

func TestSkipListRangeIndex_FindLast_Successful(t *testing.T) {
	ctx := context.Background()
	entries := []IndexEntry{
		{Key: "key1", Timestamp: time.Unix(1, 0)},
		{Key: "key2", Timestamp: time.Unix(2, 0)},
		{Key: "key3", Timestamp: time.Unix(3, 0)},
	}

	index, _ := NewSkipListRangeIndex(ctx, entries)
	last, err := index.FindLast(ctx)

	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, "key3", last.Key)
}
