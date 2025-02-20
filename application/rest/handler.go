package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Kryniol/go-moniepoint-storage-engine/application/container"
	"github.com/Kryniol/go-moniepoint-storage-engine/domain"
)

type storageHandler struct {
	c *container.Container
}

func NewStorageHandler(c *container.Container) *storageHandler {
	return &storageHandler{c: c}
}

func (h *storageHandler) Put(w http.ResponseWriter, r *http.Request) {
	h.handlePut(w, r, h.c.Engine)
}

func (h *storageHandler) Read(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	data, err := h.c.Engine.Read(r.Context(), key)
	if err != nil {
		handleFetchError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *storageHandler) ReadKeyRange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	startKey := r.URL.Query().Get("start")
	endKey := r.URL.Query().Get("end")

	if startKey == "" || endKey == "" {
		http.Error(w, "Start and End keys are required", http.StatusBadRequest)
		return
	}

	data, err := h.c.Engine.ReadKeyRange(r.Context(), startKey, endKey)
	if err != nil {
		handleFetchError(w, err)
		return
	}

	json.NewEncoder(w).Encode(data)
}

func (h *storageHandler) BatchPut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	data := map[string][]byte{}
	for key, val := range req {
		data[key] = []byte(val)
	}

	err = h.c.Engine.BatchPut(r.Context(), data)
	if err != nil {
		http.Error(w, "Failed to store batch data", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *storageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	h.handleDelete(w, r, h.c.Engine)
}

func (h *storageHandler) ReplicateReadSince(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	sinceVal := r.URL.Query().Get("since")
	if sinceVal == "" {
		http.Error(w, "Since query param is required", http.StatusBadRequest)
		return
	}

	sinceTs, err := strconv.ParseInt(sinceVal, 10, 64)
	if err != nil {
		http.Error(w, "Since query param is not a valid integer", http.StatusBadRequest)
		return
	}

	since := time.Unix(sinceTs, 0)
	data, err := h.c.Engine.ReadSince(r.Context(), &since)
	if err != nil {
		handleFetchError(w, err)
		return
	}

	json.NewEncoder(w).Encode(data)
}

func (h *storageHandler) ReplicateReadAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	data, err := h.c.Engine.ReadSince(r.Context(), nil)
	if err != nil {
		handleFetchError(w, err)
		return
	}

	json.NewEncoder(w).Encode(data)
}

func (h *storageHandler) ReplicatePut(w http.ResponseWriter, r *http.Request) {
	h.handlePut(w, r, h.c.ReplicaEngine)
}

func (h *storageHandler) ReplicateDelete(w http.ResponseWriter, r *http.Request) {
	h.handleDelete(w, r, h.c.ReplicaEngine)
}

func (h *storageHandler) handlePut(w http.ResponseWriter, r *http.Request, engine domain.StorageEngine) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Key  string `json:"key"`
		Data string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := engine.Put(r.Context(), req.Key, []byte(req.Data))
	if err != nil {
		http.Error(w, "Failed to store data", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *storageHandler) handleDelete(w http.ResponseWriter, r *http.Request, engine domain.StorageEngine) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	err := engine.Delete(r.Context(), key)
	if err != nil {
		http.Error(w, "Failed to delete key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleFetchError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrKeyNotFound) {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
}
