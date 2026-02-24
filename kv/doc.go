// Package kv определяет интерфейс [Store] для key-value хранилища.
//
// Store — единая точка входа для работы с KV: Get/Set/Delete, счётчики,
// TTL, хэши, списки и множества. Реализации находятся в дочерних пакетах:
//   - [kv/noop] — заглушка для unit-тестов
//   - [kv/redis] — реализация на базе Redis
//
// Использование:
//
//	var store kv.Store = noop.New()     // тесты
//	store, err := redis.NewDefault(cfg) // продакшн
//
// Все реализации обязаны вызывать [Store.Close] при завершении работы.
package kv
