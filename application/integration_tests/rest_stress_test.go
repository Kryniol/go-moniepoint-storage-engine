package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"testing"
	"time"
)

const (
	storeURL = "http://localhost:8080/put"
	readURL  = "http://localhost:8080/read"
	dataSize = 1_000_000
)

func TestStressAPI(t *testing.T) {
	const (
		numRequests = 1000
		numWorkers  = 10
	)

	if testing.Short() {
		t.Skip()
	}

	var wg sync.WaitGroup
	errChan := make(chan error, numRequests)

	start := time.Now()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < numRequests/numWorkers; j++ {
				key := "test_key_" + time.Now().Format("150405.000000000")
				data := randomString(dataSize)
				if err := putData(key, data); err != nil {
					errChan <- err
					continue
				}

				storedData, err := getData(key)
				if err != nil {
					errChan <- err
				}

				if storedData != data {
					errChan <- fmt.Errorf("data mismatch for key: %s", key)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			t.Errorf("Stress test failed: %v", err)
		}
	}

	elapsed := time.Since(start)
	log.Printf("Completed %d requests in %v", numRequests, elapsed)
}

func putData(key string, data string) error {
	payload, _ := json.Marshal(map[string]string{"key": key, "data": data})
	req, err := http.NewRequest(http.MethodPost, storeURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return err
	}
	return nil
}

func getData(key string) (string, error) {
	resp, err := http.Get(readURL + "?key=" + key)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func randomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	result := make([]byte, n)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}

	return string(result)
}
