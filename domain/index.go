package domain

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	maxSkipListLevel                 = 16
	skipListLevelIncreaseProbability = 0.5
)

type inMemoryKeyIndex struct {
	entries sync.Map
}

func NewInMemoryKeyIndex(entries []IndexEntry) *inMemoryKeyIndex {
	index := &inMemoryKeyIndex{}
	for _, entry := range entries {
		index.entries.Store(entry.Key, entry)
	}

	return index
}

func (k *inMemoryKeyIndex) Find(_ context.Context, key string) (*IndexEntry, error) {
	res, ok := k.entries.Load(key)
	if !ok {
		return nil, ErrKeyNotFound
	}

	entry := res.(IndexEntry)

	return &entry, nil
}

func (k *inMemoryKeyIndex) Add(_ context.Context, entry IndexEntry) error {
	k.entries.Store(entry.Key, entry)

	return nil
}

type skipListNode struct {
	entry IndexEntry
	next  []*skipListNode
}

type skipListRangeIndex struct {
	head  *skipListNode
	level int
	lock  sync.RWMutex
}

func NewSkipListRangeIndex(ctx context.Context, entries []IndexEntry) (*skipListRangeIndex, error) {
	index := &skipListRangeIndex{
		head: &skipListNode{next: make([]*skipListNode, maxSkipListLevel)},
	}
	for _, entry := range entries {
		err := index.Add(ctx, entry)
		if err != nil {
			return nil, fmt.Errorf("failed to add entry to skip list at initialization: %w", err)
		}
	}

	return index, nil
}

func (k *skipListRangeIndex) Add(_ context.Context, entry IndexEntry) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	update := make([]*skipListNode, maxSkipListLevel)
	curr := k.head

	for i := k.level - 1; i >= 0; i-- {
		for curr.next[i] != nil && curr.next[i].entry.Timestamp.Before(entry.Timestamp) {
			curr = curr.next[i]
		}
		update[i] = curr
	}

	level := randomLevel()
	if level > k.level {
		for i := k.level; i < level; i++ {
			update[i] = k.head
		}
		k.level = level
	}

	newNode := &skipListNode{
		entry: entry,
		next:  make([]*skipListNode, level),
	}

	for i := 0; i < level; i++ {
		newNode.next[i] = update[i].next[i]
		update[i].next[i] = newNode
	}

	return nil
}

func (k *skipListRangeIndex) FindRange(_ context.Context, from, to IndexEntry) (map[string]IndexEntry, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()

	result := map[string]IndexEntry{}
	curr := k.head

	for i := k.level - 1; i >= 0; i-- {
		for curr.next[i] != nil && curr.next[i].entry.Timestamp.Before(from.Timestamp) {
			curr = curr.next[i]
		}
	}

	curr = curr.next[0]
	for curr != nil && (curr.entry.Timestamp.Before(to.Timestamp) || curr.entry.Timestamp.Equal(to.Timestamp)) {
		result[curr.entry.Key] = curr.entry
		curr = curr.next[0]
	}

	return result, nil
}

func (k *skipListRangeIndex) FindSince(_ context.Context, since time.Time) (map[string]IndexEntry, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()

	result := make(map[string]IndexEntry)
	curr := k.head

	for i := k.level - 1; i >= 0; i-- {
		for curr.next[i] != nil && curr.next[i].entry.Timestamp.Before(since) {
			curr = curr.next[i]
		}
	}

	curr = curr.next[0]
	for curr != nil {
		if curr.entry.Timestamp.Before(since) {
			curr = curr.next[0]
			continue
		}

		result[curr.entry.Key] = curr.entry
		curr = curr.next[0]
	}

	return result, nil
}

func (k *skipListRangeIndex) FindFirst(context.Context) (*IndexEntry, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()

	if k.head.next[0] == nil {
		return nil, nil
	}

	return &k.head.next[0].entry, nil
}

func (k *skipListRangeIndex) FindLast(context.Context) (*IndexEntry, error) {
	k.lock.RLock()
	defer k.lock.RUnlock()

	if k.head.next[0] == nil {
		return nil, nil
	}

	curr := k.head
	for curr.next[0] != nil {
		curr = curr.next[0]
	}

	return &curr.entry, nil
}

func randomLevel() int {
	level := 1
	for rand.Float64() < skipListLevelIncreaseProbability && level < maxSkipListLevel {
		level++
	}
	return level
}
