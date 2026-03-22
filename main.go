package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type KVStore struct {
	mu    sync.RWMutex
	store map[string]string
}

func NewKVStore() *KVStore {
	return &KVStore{
		store: make(map[string]string),
	}
}

func (kv *KVStore) Set(key, value string) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.store[key] = value
}

func (kv *KVStore) Get(key string) (string, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	val, ok := kv.store[key]
	return val, ok
}

func (kv *KVStore) Delete(key string) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if _, exists := kv.store[key]; !exists {
		panic(fmt.Sprintf("key '%s' does not exist", key))
	}
	delete(kv.store, key)
}

func (kv *KVStore) GetAll() map[string]string {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	copyMap := make(map[string]string)
	for k, v := range kv.store {
		copyMap[k] = v
	}
	return copyMap
}

type Item struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var store = NewKVStore()

func Set(w http.ResponseWriter, r *http.Request) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	store.Set(item.Key, item.Value)
	w.WriteHeader(http.StatusCreated)
}

func Get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	val, ok := store.Get(key)
	if !ok {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"value": val})
}

func Delete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	defer func() {
		if rec := recover(); rec != nil {
			http.Error(w, fmt.Sprintf("%v", rec), http.StatusInternalServerError)
		}
	}()

	store.Delete(key)
	w.WriteHeader(http.StatusNoContent)
}

func setupRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /item", Set)
	mux.HandleFunc("GET /item/{key}", Get)
	mux.HandleFunc("DELETE /item/{key}", Delete)
	return mux
}

func main() {
	fmt.Println("Starting server on :8080")
	http.ListenAndServe(":8080", setupRouter())
}
