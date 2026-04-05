package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

type Storage interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string)
	GetAll() (map[string]string, error)
}

type KVStore struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewKVStore() *KVStore {
	return &KVStore{store: make(map[string]string)}
}

func (kv *KVStore) Set(key, value string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.store[key] = value
	return nil
}

func (kv *KVStore) Get(key string) (string, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	val, ok := kv.store[key]
	if !ok {
		return "", errors.New("item not found")
	}
	return val, nil
}

func (kv *KVStore) Delete(key string) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if _, exists := kv.store[key]; !exists {
		panic(fmt.Sprintf("key '%s' does not exist", key))
	}
	delete(kv.store, key)
}

func (kv *KVStore) GetAll() (map[string]string, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	copyMap := make(map[string]string)
	for k, v := range kv.store {
		copyMap[k] = v
	}
	return copyMap, nil
}

type Item struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Server struct {
	store Storage
}

func NewServer(s Storage) *Server {
	return &Server{store: s}
}

func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if err := s.store.Set(item.Key, item.Value); err != nil {
		http.Error(w, "Failed to save", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	val, err := s.store.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"value": val})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	defer func() {
		if rec := recover(); rec != nil {
			http.Error(w, fmt.Sprintf("%v", rec), http.StatusInternalServerError)
		}
	}()

	s.store.Delete(key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePop(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	val, err := s.store.Get(key)
	if err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	defer func() {
		if rec := recover(); rec != nil {
			http.Error(w, fmt.Sprintf("%v", rec), http.StatusInternalServerError)
		}
	}()

	s.store.Delete(key)
	json.NewEncoder(w).Encode(map[string]string{"value": val})
}

func setupRouter(s Storage) *http.ServeMux {
	server := NewServer(s)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /item", server.handleSet)
	mux.HandleFunc("GET /item/{key}", server.handleGet)
	mux.HandleFunc("DELETE /item/{key}", server.handleDelete)
	mux.HandleFunc("POST /item/pop/{key}", server.handlePop)
	return mux
}

func main() {
	store := NewKVStore()
	fmt.Println("Starting server on :8080")
	http.ListenAndServe(":8080", setupRouter(store))
}
