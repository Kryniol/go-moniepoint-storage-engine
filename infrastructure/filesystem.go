package infrastructure

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Kryniol/go-moniepoint-storage-engine/domain"
)

const maxFileSize = 1024 * 1024 * 1024 // 1GB

type filesystemWriteStorage struct {
	index          domain.KeyIndex
	dirPath        string
	activeFh       *os.File
	activeFileName string
	mutex          *sync.Mutex
	now            func() time.Time
	currentSize    int64
}

func NewFilesystemWriteStorage(
	index domain.KeyIndex,
	dirPath string,
	now func() time.Time,
) (*filesystemWriteStorage, error) {
	storage := &filesystemWriteStorage{index: index, dirPath: dirPath, mutex: &sync.Mutex{}, now: now}
	err := storage.openLatestFile()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize data file: %w", err)
	}

	return storage, nil
}

func (s *filesystemWriteStorage) Set(ctx context.Context, key string, data []byte) error {
	now := s.now()
	keyBytes := []byte(key)
	dataLen := int64(len(data))
	header := fmt.Sprintf("%d;%d;%d;%s;", now.Unix(), len(keyBytes), dataLen, key)
	headerLen := int64(len([]byte(header)))

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.currentSize+headerLen+dataLen >= maxFileSize {
		err := s.rotateFile()
		if err != nil {
			return fmt.Errorf("failed to rotate file: %w", err)
		}
	}

	currentPos, _ := s.activeFh.Seek(0, 2)
	_, err := s.activeFh.WriteString(header)
	if err != nil {
		return fmt.Errorf("failed to write header for key %s: %w", key, err)
	}

	_, err = s.activeFh.Write(data)
	_, err = s.activeFh.WriteString("\n")
	if err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	err = s.index.Add(
		ctx, domain.IndexEntry{
			Key:           key,
			FileID:        s.activeFileName,
			ValueSize:     dataLen,
			ValuePosition: currentPos + headerLen,
			Timestamp:     now,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to add entry for key %s to index: %w", key, err)
	}

	s.currentSize += headerLen + dataLen + 1

	return nil
}

func (s *filesystemWriteStorage) Delete(ctx context.Context, key string) error {
	err := s.Set(ctx, key, []byte(""))
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

func (s *filesystemWriteStorage) openLatestFile() error {
	files, err := os.ReadDir(s.dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}

	var latestTimestamp int64
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasSuffix(name, ".dat") {
			continue
		}

		timestampStr := name[:len(name)-4]
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			continue // ignore invalid timestamps
		}

		if timestamp > latestTimestamp {
			latestTimestamp = timestamp
		}
	}

	// ff no valid file was found, rotate to create new file
	if latestTimestamp == 0 {
		err = s.rotateFile()
		if err != nil {
			return fmt.Errorf("failed to rotate file: %w", err)
		}

		return nil
	}

	fileName := fmt.Sprintf("%d.dat", latestTimestamp)
	filePath := filepath.Join(s.dirPath, fileName)
	fh, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return fmt.Errorf("failed to open latest file %s: %w", filePath, err)
	}

	s.activeFh = fh
	s.activeFileName = fileName
	s.currentSize, _ = fh.Seek(0, 2)

	return nil
}

func (s *filesystemWriteStorage) rotateFile() error {
	if s.activeFh != nil {
		_ = s.activeFh.Close()
	}

	newFileName := fmt.Sprintf("%d.dat", s.now().Unix())
	filePath := filepath.Join(s.dirPath, newFileName)

	fh, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create new file: %w", err)
	}

	s.activeFh = fh
	s.activeFileName = newFileName
	s.currentSize = 0
	log.Printf("rotated to new file: %s", filePath)

	return nil
}

type filesystemReadStorage struct {
	index       domain.Index
	fileHandles sync.Map
	// fileHandles map[string]*os.File
	dirPath string
}

func NewFilesystemReadStorage(index domain.Index, dirPath string) *filesystemReadStorage {
	return &filesystemReadStorage{index: index, dirPath: dirPath}
}

func (s *filesystemReadStorage) Get(ctx context.Context, key string) ([]byte, error) {
	keyDirEntry, err := s.index.Find(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to find key %s in index: %w", key, err)
	}

	return s.doGet(*keyDirEntry)
}

func (s *filesystemReadStorage) GetRange(ctx context.Context, startKey, endKey string) (map[string][]byte, error) {
	entries, err := s.index.FindRange(ctx, startKey, endKey)
	if err != nil {
		return nil, fmt.Errorf("failed to find key range in index: %w", err)
	}

	result, err := s.doGetMany(entries)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys for range: %w", err)
	}

	return result, nil
}

func (s *filesystemReadStorage) GetSince(ctx context.Context, since time.Time) (map[string][]byte, error) {
	entries, err := s.index.FindSince(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to find key range in index: %w", err)
	}

	result, err := s.doGetMany(entries)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys for since query: %w", err)
	}

	return result, nil
}

func (s *filesystemReadStorage) GetLastTimestamp(ctx context.Context) (*time.Time, error) {
	last, err := s.index.FindLast(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find last key in index: %w", err)
	}

	if last == nil {
		return nil, nil
	}

	return &last.Timestamp, nil
}

func (s *filesystemReadStorage) GetAll(ctx context.Context) (map[string][]byte, error) {
	first, err := s.index.FindFirst(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find first key in index: %w", err)
	}

	if first == nil {
		return nil, nil
	}

	last, err := s.index.FindLast(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find last key in index: %w", err)
	}

	entries, err := s.index.FindRange(ctx, first.Key, last.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to find key range in index: %w", err)
	}

	result, err := s.doGetMany(entries)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys for range: %w", err)
	}

	return result, nil
}

func (s *filesystemReadStorage) doGet(keyDirEntry domain.IndexEntry) ([]byte, error) {
	if keyDirEntry.ValueSize == 0 {
		return nil, domain.ErrKeyNotFound
	}

	var (
		fh  *os.File
		err error
	)

	fhDirty, ok := s.fileHandles.Load(keyDirEntry.FileID)
	if !ok {
		filePath := filepath.Join(s.dirPath, keyDirEntry.FileID)
		fh, err = os.Open(filePath) // read only
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
		}

		s.fileHandles.Store(keyDirEntry.FileID, fh)
	} else {
		fh = fhDirty.(*os.File)
	}

	val := make([]byte, keyDirEntry.ValueSize)
	_, err = fh.ReadAt(val, keyDirEntry.ValuePosition)

	return val, nil
}

func (s *filesystemReadStorage) doGetMany(entries map[string]domain.IndexEntry) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for _, entry := range entries {
		val, err := s.doGet(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to read key %s: %w", entry.Key, err)
		}

		result[entry.Key] = val
	}

	return result, nil
}

type filesystemKeydirIndexer struct {
	dirPath string
}

func NewFilesystemKeydirIndexer(dirPath string) *filesystemKeydirIndexer {
	return &filesystemKeydirIndexer{dirPath: dirPath}
}

func (i *filesystemKeydirIndexer) Reindex(context.Context) ([]domain.IndexEntry, error) {
	entriesCh := make(chan domain.IndexEntry)
	var wg sync.WaitGroup
	files, err := os.ReadDir(i.dirPath)
	if err != nil {
		close(entriesCh)
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		wg.Add(1)
		go func(filename string) {
			defer wg.Done()
			i.reindexFile(filename, entriesCh)
		}(file.Name())
	}

	go func() {
		wg.Wait()
		close(entriesCh)
	}()

	// keep only the latest value for each key
	var entriesMap = make(map[string]domain.IndexEntry)
	for entry := range entriesCh {
		if existingEntry, exists := entriesMap[entry.Key]; !exists || entry.Timestamp.After(existingEntry.Timestamp) {
			entriesMap[entry.Key] = entry
		}
	}

	var entries []domain.IndexEntry
	for _, entry := range entriesMap {
		entries = append(entries, entry)
	}

	return entries, nil
}

func (i *filesystemKeydirIndexer) reindexFile(fileName string, out chan<- domain.IndexEntry) {
	f, err := os.Open(filepath.Join(i.dirPath, fileName))
	if err != nil {
		return
	}
	defer f.Close()

	var currentPos int64
	scanner := bufio.NewScanner(f)
	movePastLine := func(line string) {
		currentPos += int64(len(line)) + 1
	}

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ";")
		if len(parts) != 5 {
			movePastLine(line)
			continue
		}

		timestampInt, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			movePastLine(line)
			continue
		}

		timestamp := time.Unix(timestampInt, 0)
		valueSize, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			movePastLine(line)
			continue
		}

		key := parts[3]
		valuePosition := currentPos + int64(len(parts[0])+len(parts[1])+len(parts[2])+len(parts[3])+4) // add length of 4 semicolons

		out <- domain.IndexEntry{
			Key:           key,
			FileID:        fileName,
			ValueSize:     valueSize,
			ValuePosition: valuePosition,
			Timestamp:     timestamp,
		}

		currentPos += int64(len(line)) + 1
	}
}
