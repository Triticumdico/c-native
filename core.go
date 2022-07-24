package main

import (
	"errors"
	"sync"
)

var store = struct {
	sync.RWMutex
	m map[string]string
}{m: make(map[string]string)}

var ErrorNoSuchKey = errors.New("no such key")

func Put(key string, value string) error {
	store.Lock() // Take a write lock
	store.m[key] = value
	store.Unlock() // Release the write lock

	return nil
}

func Get(key string) (string, error) {
	store.RLock() // Take a read lock
	value, ok := store.m[key]
	store.RUnlock() // Release a read lock

	if !ok {
		return "", ErrorNoSuchKey
	}

	return value, nil
}

func Delete(key string) error {
	store.Lock() // Take a write lock
	delete(store.m, key)
	store.Unlock() // Release a write lock

	return nil
}
