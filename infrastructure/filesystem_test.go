package infrastructure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kryniol/go-moniepoint-storage-engine/domain"
	domainmocks "github.com/Kryniol/go-moniepoint-storage-engine/mocks/domain"
)

const testDir = "./testdata"

func TestNewFilesystemWriteStorage_NoDataFiles_CreatesFile(t *testing.T) {
	mockIndex := new(domainmocks.MockKeyIndex)
	nowFunc := func() time.Time { return time.Unix(1620000000, 0) }
	dirPath := setupTestDir(t)

	storage, err := NewFilesystemWriteStorage(mockIndex, dirPath, nowFunc)

	require.NoError(t, err)
	assert.NotNil(t, storage.activeFh)
	assert.Equal(t, "1620000000.dat", storage.activeFileName)
}

func TestNewFilesystemWriteStorage_OpensLatestFile(t *testing.T) {
	nowFunc := func() time.Time { return time.Unix(1620000004, 0) }
	dirPath := setupTestDir(t)

	existingFileName := fmt.Sprintf("%d.dat", 1620000001)
	existingFilePath := filepath.Join(dirPath, existingFileName)
	_, err := os.Create(existingFilePath)
	require.NoError(t, err)

	storage, err := NewFilesystemWriteStorage(domainmocks.NewMockKeyIndex(t), dirPath, nowFunc)
	require.NoError(t, err)

	assert.Equal(t, existingFileName, storage.activeFileName)
}

func TestFilesystemWriteStorage_Set_Successful(t *testing.T) {
	ctx := context.Background()
	nowFunc := func() time.Time { return time.Unix(1620000001, 0) }
	dirPath := setupTestDir(t)
	key := "testKey"
	data := []byte("testData")

	mockIndex := domainmocks.NewMockKeyIndex(t)
	mockIndex.On(
		"Add", ctx, mock.MatchedBy(
			func(entry domain.IndexEntry) bool {
				return entry.Key == key && entry.ValueSize == int64(len(data))
			},
		),
	).Return(nil)

	storage, err := NewFilesystemWriteStorage(mockIndex, dirPath, nowFunc)
	require.NoError(t, err)

	err = storage.Set(ctx, key, data)

	require.NoError(t, err)
	fileContent, err := os.ReadFile(filepath.Join(dirPath, storage.activeFileName))
	require.NoError(t, err)
	expectedHeader := fmt.Sprintf("%d;%d;%d;%s;", nowFunc().Unix(), len(key), len(data), key)
	assert.Contains(t, string(fileContent), expectedHeader+"testData\n")
}

func TestFilesystemWriteStorage_Set_RotatesFileWhenFull(t *testing.T) {
	ctx := context.Background()
	nowFunc := func() time.Time { return time.Unix(1620000002, 0) }
	dirPath := setupTestDir(t)
	key := "newKey"
	data := []byte("newData")

	mockIndex := domainmocks.NewMockKeyIndex(t)
	mockIndex.On("Add", ctx, mock.Anything).Return(nil)

	storage, err := NewFilesystemWriteStorage(mockIndex, dirPath, nowFunc)
	require.NoError(t, err)
	storage.currentSize = maxFileSize - 1

	err = storage.Set(ctx, key, data)

	require.NoError(t, err)
	assert.Equal(t, "1620000002.dat", storage.activeFileName)
}

func TestFilesystemWriteStorage_Delete_Successful(t *testing.T) {
	ctx := context.Background()
	nowFunc := func() time.Time { return time.Unix(1620000003, 0) }
	dirPath := setupTestDir(t)
	key := "deleteKey"

	mockIndex := domainmocks.NewMockKeyIndex(t)
	mockIndex.On("Add", ctx, mock.Anything).Return(nil)

	storage, err := NewFilesystemWriteStorage(mockIndex, dirPath, nowFunc)
	require.NoError(t, err)

	err = storage.Delete(ctx, key)

	require.NoError(t, err)
	fileContent, err := os.ReadFile(filepath.Join(dirPath, storage.activeFileName))
	require.NoError(t, err)
	expectedHeader := fmt.Sprintf("%d;%d;0;%s;", nowFunc().Unix(), len(key), key)
	assert.Contains(t, string(fileContent), expectedHeader+"\n")
}

func TestFilesystemReadStorage_Get_Successful(t *testing.T) {
	ctx := context.Background()
	dirPath := setupTestDir(t)
	fileID := "testfile.dat"
	key := "testKey"
	value := "testValue"
	timestamp := time.Now()
	formattedEntry := formatDataEntry(timestamp, key, value)
	mockEntry := domain.IndexEntry{
		Key:           key,
		FileID:        fileID,
		ValueSize:     int64(len(value)),
		ValuePosition: int64(len(formattedEntry) - len(value) - 1),
		Timestamp:     timestamp,
	}
	createTestFile(t, dirPath, fileID, formattedEntry)

	mockIndex := domainmocks.NewMockIndex(t)
	mockIndex.On("Find", ctx, key).Return(&mockEntry, nil)

	storage := NewFilesystemReadStorage(mockIndex, dirPath)
	data, err := storage.Get(ctx, key)

	require.NoError(t, err)
	assert.Equal(t, []byte(value), data)
}

func TestFilesystemReadStorage_GetRange_Successful(t *testing.T) {
	ctx := context.Background()
	dirPath := setupTestDir(t)
	fileID := "rangeFile.dat"
	timestamp := time.Now()
	key1, value1 := "key1", "value1"
	key2, value2 := "key2", "value2"
	formattedData1 := formatDataEntry(timestamp, key1, value1)
	formattedData2 := formatDataEntry(timestamp, key2, value2)
	formattedData := formattedData1 + formattedData2
	entries := map[string]domain.IndexEntry{
		"key1": {
			Key: "key1", FileID: fileID, ValueSize: int64(len(value1)),
			ValuePosition: int64(len(formattedData1) - len(value1) - 1),
		},
		"key2": {
			Key: "key2", FileID: fileID, ValueSize: int64(len(value2)),
			ValuePosition: int64(len(formattedData) - len(value2) - 1),
		},
	}
	createTestFile(t, dirPath, fileID, formattedData)

	mockIndex := domainmocks.NewMockIndex(t)
	mockIndex.On("FindRange", ctx, "key1", "key2").Return(entries, nil)

	storage := NewFilesystemReadStorage(mockIndex, dirPath)
	data, err := storage.GetRange(ctx, "key1", "key2")

	require.NoError(t, err)
	assert.Equal(
		t, map[string][]byte{
			"key1": []byte(value1),
			"key2": []byte(value2),
		}, data,
	)
}

func TestFilesystemReadStorage_GetSince_Successful(t *testing.T) {
	ctx := context.Background()
	dirPath := setupTestDir(t)
	fileID := "sinceFile.dat"
	timestamp := time.Now().Add(-time.Hour)
	key, value := "key1", "sinceData"
	formattedData := formatDataEntry(timestamp, key, value)
	entries := map[string]domain.IndexEntry{
		"key1": {
			Key: key, FileID: fileID, ValueSize: int64(len(value)),
			ValuePosition: int64(len(formattedData) - len(value) - 1), Timestamp: timestamp,
		},
	}
	createTestFile(t, dirPath, fileID, formattedData)

	mockIndex := domainmocks.NewMockIndex(t)
	mockIndex.On("FindSince", ctx, timestamp).Return(entries, nil)

	storage := NewFilesystemReadStorage(mockIndex, dirPath)
	data, err := storage.GetSince(ctx, timestamp)

	require.NoError(t, err)
	assert.Equal(
		t, map[string][]byte{
			"key1": []byte(value),
		}, data,
	)
}

func TestFilesystemReadStorage_GetLastTimestamp_Successful(t *testing.T) {
	ctx := context.Background()
	dirPath := setupTestDir(t)

	lastTimestamp := time.Now()
	mockEntry := &domain.IndexEntry{Timestamp: lastTimestamp}

	mockIndex := domainmocks.NewMockIndex(t)
	mockIndex.On("FindLast", ctx).Return(mockEntry, nil)

	storage := NewFilesystemReadStorage(mockIndex, dirPath)
	timestamp, err := storage.GetLastTimestamp(ctx)

	require.NoError(t, err)
	assert.Equal(t, lastTimestamp, *timestamp)
}

func TestFilesystemReadStorage_GetAll_Successful(t *testing.T) {
	ctx := context.Background()
	dirPath := setupTestDir(t)
	fileID := "allFile.dat"
	timestamp := time.Now()
	key1, value1 := "key1", "allData1"
	key2, value2 := "key2", "allData2"
	formattedData1 := formatDataEntry(timestamp, key1, value1)
	formattedData2 := formatDataEntry(timestamp, key2, value2)
	formattedData := formattedData1 + formattedData2
	mockFirstEntry := domain.IndexEntry{
		Key: "key1", FileID: fileID, ValueSize: int64(len(value1)),
		ValuePosition: int64(len(formattedData1) - len(value1) - 1),
	}
	mockLastEntry := domain.IndexEntry{
		Key: "key2", FileID: fileID, ValueSize: int64(len(value2)),
		ValuePosition: int64(len(formattedData) - len(value2) - 1),
	}
	entries := map[string]domain.IndexEntry{
		"key1": mockFirstEntry,
		"key2": mockLastEntry,
	}
	createTestFile(t, dirPath, fileID, formattedData)

	mockIndex := domainmocks.NewMockIndex(t)
	mockIndex.On("FindFirst", ctx).Return(&mockFirstEntry, nil)
	mockIndex.On("FindLast", ctx).Return(&mockLastEntry, nil)
	mockIndex.On("FindRange", ctx, key1, key2).Return(entries, nil)

	storage := NewFilesystemReadStorage(mockIndex, dirPath)
	data, err := storage.GetAll(ctx)

	require.NoError(t, err)
	assert.Equal(
		t, map[string][]byte{
			"key1": []byte(value1),
			"key2": []byte(value2),
		}, data,
	)
}

func TestFilesystemKeydirIndexer_Reindex_SingleFile(t *testing.T) {
	dirPath := setupTestDir(t)
	fileName := "testfile.dat"
	timestamp1 := time.Now().Add(-time.Hour)
	timestamp2 := time.Now()
	key := "testKey"
	oldEntry := formatDataEntry(timestamp1, key, "oldValue")
	newEntry := formatDataEntry(timestamp2, key, "newValue")
	createTestFile(t, dirPath, fileName, oldEntry+newEntry)

	indexer := NewFilesystemKeydirIndexer(dirPath)
	entries, err := indexer.Reindex(context.Background())

	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, key, entries[0].Key)
	assert.Equal(t, fileName, entries[0].FileID)
	assert.Equal(t, timestamp2.Unix(), entries[0].Timestamp.Unix())
	assert.Equal(t, int64(len("newValue")), entries[0].ValueSize)
}

func TestFilesystemKeydirIndexer_Reindex_MultipleFiles(t *testing.T) {
	dirPath := setupTestDir(t)
	file1 := "file1.dat"
	file2 := "file2.dat"
	timestamp1 := time.Now().Add(-2 * time.Hour)
	timestamp2 := time.Now().Add(-time.Hour)
	timestamp3 := time.Now()
	key1, key2 := "key1", "key2"
	content1 := formatDataEntry(timestamp1, key1, "value1") + formatDataEntry(timestamp2, key2, "value2")
	content2 := formatDataEntry(timestamp3, key1, "latestValue1")
	createTestFile(t, dirPath, file1, content1)
	createTestFile(t, dirPath, file2, content2)
	expected := map[string]domain.IndexEntry{
		key1: {Key: key1, FileID: file2, Timestamp: timestamp3, ValueSize: int64(len("latestValue1"))},
		key2: {Key: key2, FileID: file1, Timestamp: timestamp2, ValueSize: int64(len("value2"))},
	}

	indexer := NewFilesystemKeydirIndexer(dirPath)
	entries, err := indexer.Reindex(context.Background())

	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, entry := range entries {
		expectedEntry, exists := expected[entry.Key]
		assert.True(t, exists)
		assert.Equal(t, expectedEntry.FileID, entry.FileID)
		assert.Equal(t, expectedEntry.Timestamp.Unix(), entry.Timestamp.Unix())
		assert.Equal(t, expectedEntry.ValueSize, entry.ValueSize)
	}
}

func TestFilesystemKeydirIndexer_Reindex_EmptyDirectory_NoEntriesAreReturned(t *testing.T) {
	dirPath := setupTestDir(t)

	indexer := NewFilesystemKeydirIndexer(dirPath)
	entries, err := indexer.Reindex(context.Background())

	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestFilesystemKeydirIndexer_Reindex_FileParsingError_NoEntriesAreReturned(t *testing.T) {
	dirPath := setupTestDir(t)
	invalidContent := "invalid;data;not;correct\n" // Incorrect format
	createTestFile(t, dirPath, "corrupt.dat", invalidContent)

	indexer := NewFilesystemKeydirIndexer(dirPath)
	entries, err := indexer.Reindex(context.Background())

	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestFilesystemKeydirIndexer_Reindex_HandlesConcurrentFiles(t *testing.T) {
	dirPath := setupTestDir(t)
	timestamp := time.Now()
	keyA, keyB := "keyA", "keyB"
	valueA, valueB := "valueA", "valueB"
	createTestFile(t, dirPath, "fileA.dat", formatDataEntry(timestamp, keyA, valueA))
	createTestFile(t, dirPath, "fileB.dat", formatDataEntry(timestamp, keyB, valueB))
	expectedKeys := map[string]string{
		keyA: valueA,
		keyB: valueB,
	}

	indexer := NewFilesystemKeydirIndexer(dirPath)
	entries, err := indexer.Reindex(context.Background())

	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, entry := range entries {
		expectedValue, exists := expectedKeys[entry.Key]
		assert.True(t, exists)
		assert.Equal(t, int64(len(expectedValue)), entry.ValueSize)
	}
}

func setupTestDir(t *testing.T) string {
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	t.Cleanup(
		func() {
			os.RemoveAll(testDir)
		},
	)

	return testDir
}

func createTestFile(t *testing.T, dirPath, fileName string, content string) {
	filePath := filepath.Join(dirPath, fileName)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
}

func formatDataEntry(timestamp time.Time, key string, value string) string {
	header := fmt.Sprintf("%d;%d;%d;%s;", timestamp.Unix(), len(key), len(value), key)
	return header + value + "\n"
}
