package integration_tests

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Kryniol/go-moniepoint-storage-engine/application/container"
	"github.com/Kryniol/go-moniepoint-storage-engine/config"
	"github.com/Kryniol/go-moniepoint-storage-engine/domain"
)

var engine domain.StorageEngine

func TestMain(m *testing.M) {
	os.Exit(doRun(m))
}

func doRun(m *testing.M) int {
	curDir := getCurrentDir()
	dataDir := filepath.Join(curDir, "test_data")
	err := os.MkdirAll(dataDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dataDir)

	cfg := config.Config{DataPath: dataDir, Mode: config.Leader}
	ctx := context.Background()
	c, err := container.NewServiceContainer(ctx, cfg)
	if err != nil {
		panic(err)
	}

	engine = c.Engine

	return m.Run()
}

func BenchmarkStorageEngine_PutParallel(b *testing.B) {
	ctx := context.Background()
	data := randomBytes(256)

	b.RunParallel(
		func(pb *testing.PB) {
			for pb.Next() {
				key := suffix("test_put", rand.Int())
				if err := engine.Put(ctx, key, data); err != nil {
					b.Fatal(err)
				}
			}
		},
	)
}

func BenchmarkStorageEngine_ReadParallel(b *testing.B) {
	ctx := context.Background()
	key := "benchmark_read"
	data := randomBytes(256)

	if err := engine.Put(ctx, key, data); err != nil {
		b.Fatalf("setup failed: %v", err)
	}

	b.RunParallel(
		func(pb *testing.PB) {
			for pb.Next() {
				_, err := engine.Read(ctx, key)
				if err != nil {
					b.Fatal(err)
				}
			}
		},
	)
}

func BenchmarkStorageEngine_ReadKeyRangeParallel(b *testing.B) {
	ctx := context.Background()
	startKey, endKey := "range_key_0", "range_key_99"
	for i := 0; i < 100; i++ {
		key := suffix("range_key", i)
		if err := engine.Put(ctx, key, randomBytes(256)); err != nil {
			b.Fatalf("setup failed: %v", err)
		}
	}

	b.RunParallel(
		func(pb *testing.PB) {
			for pb.Next() {
				_, err := engine.ReadKeyRange(ctx, startKey, endKey)
				if err != nil {
					b.Fatal(err)
				}
			}
		},
	)
}

func BenchmarkStorageEngine_BatchPutParallel(b *testing.B) {
	ctx := context.Background()

	b.RunParallel(
		func(pb *testing.PB) {
			for pb.Next() {
				batch := make(map[string][]byte, 10)
				for j := 0; j < 10; j++ {
					batch[suffix("batch", rand.Int())] = randomBytes(256)
				}

				if err := engine.BatchPut(ctx, batch); err != nil {
					b.Fatal(err)
				}
			}
		},
	)
}

func BenchmarkStorageEngine_DeleteParallel(b *testing.B) {
	ctx := context.Background()
	key := "benchmark_delete"
	data := randomBytes(256)

	if err := engine.Put(ctx, key, data); err != nil {
		b.Fatalf("setup failed: %v", err)
	}

	b.RunParallel(
		func(pb *testing.PB) {
			for pb.Next() {
				if err := engine.Delete(ctx, key); err != nil {
					b.Fatal(err)
				}
				// Reinsert key to ensure deletion can be tested repeatedly
				if err := engine.Put(ctx, key, data); err != nil {
					b.Fatal(err)
				}
			}
		},
	)
}

func getCurrentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return dir
}

func randomBytes(size int) []byte {
	b := make([]byte, size)
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	r.Read(b)

	return b
}

func suffix(val string, i int) string {
	return val + "_" + strconv.Itoa(i)
}
