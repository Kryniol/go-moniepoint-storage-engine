package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type restWriteStorage struct {
	c       *http.Client
	baseURL string
}

func NewRESTWriteStorage(baseURL string) *restWriteStorage {
	return &restWriteStorage{c: http.DefaultClient, baseURL: baseURL}
}

func (s *restWriteStorage) Set(ctx context.Context, key string, data []byte) error {
	uri := fmt.Sprintf("%s/put", s.baseURL)
	reqBody, err := json.Marshal(
		map[string]string{
			"key":  key,
			"data": string(data),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (s *restWriteStorage) Delete(ctx context.Context, key string) error {
	uri := fmt.Sprintf("%s/delete?key=%s", s.baseURL, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, uri, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

type restReplicaFanoutWriteStorage struct {
	hosts      []string
	writeQueue sync.Map
	mu         sync.Mutex
}

func NewRESTReplicaFanoutWriteStorage(hosts []string) *restReplicaFanoutWriteStorage {
	return &restReplicaFanoutWriteStorage{hosts: hosts}
}

func (s *restReplicaFanoutWriteStorage) Set(ctx context.Context, key string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.writeQueue.Store(key, data)
	go s.processWrite(ctx, key)

	return nil
}

func (s *restReplicaFanoutWriteStorage) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := "/delete?key=" + url.QueryEscape(key)
	return s.fanoutRequest(ctx, endpoint, nil)
}

func (s *restReplicaFanoutWriteStorage) processWrite(ctx context.Context, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, ok := s.writeQueue.Load(key)
	if !ok {
		return
	}

	type setRequest struct {
		Key  string `json:"key"`
		Data string `json:"data"`
	}

	payload := setRequest{Key: key, Data: string(data.([]byte))}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		log.Println("JSON Marshal error:", err)
		return
	}

	err = s.fanoutRequest(ctx, "/replicate/put", jsonBody)
	if err != nil {
		log.Println("Write error:", err)
	}

	s.writeQueue.Delete(key)
}

func (s *restReplicaFanoutWriteStorage) fanoutRequest(ctx context.Context, endpoint string, body []byte) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(s.hosts))

	for _, host := range s.hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			uri := host + endpoint
			if err := retryRequest(ctx, uri, http.MethodPost, body); err != nil {
				errCh <- fmt.Errorf("host %s: %w", host, err)
			}
		}(host)
	}

	wg.Wait()
	close(errCh)

	var allErrors []error
	for err := range errCh {
		allErrors = append(allErrors, err)
	}

	return errors.Join(allErrors...)
}

type leaderRESTDataFetcher struct {
	c       *http.Client
	baseURL string
}

func NewLeaderRESTDataFetcher(baseURL string) *leaderRESTDataFetcher {
	return &leaderRESTDataFetcher{
		baseURL: baseURL,
		c:       &http.Client{},
	}
}

func (f *leaderRESTDataFetcher) FetchSince(ctx context.Context, since time.Time) (map[string][]byte, error) {
	uri := fmt.Sprintf("%s/replicate/readsince?since=%d", f.baseURL, since.Unix())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return f.doFetchData(req)
}

func (f *leaderRESTDataFetcher) FetchAll(ctx context.Context) (map[string][]byte, error) {
	uri := fmt.Sprintf("%s/replicate/readall", f.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return f.doFetchData(req)
}

func (f *leaderRESTDataFetcher) doFetchData(req *http.Request) (map[string][]byte, error) {
	resp, err := f.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data map[string][]byte
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return data, nil
}

func retryRequest(_ context.Context, url, method string, body []byte) error {
	const (
		timeout        = 5 * time.Second
		maxRetries     = 3
		initialBackoff = time.Millisecond * 100
	)

	client := &http.Client{Timeout: timeout}
	backoff := initialBackoff
	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest(method, url, bytes.NewReader(body))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			return nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		select {
		case <-time.After(backoff):
			backoff *= 2
		}
	}
	return fmt.Errorf("request to %s failed after %d retries", url, maxRetries)
}
