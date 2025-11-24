package agent

import (
	"sync"
	"testing"
)

// TestNewRandConcurrency 测试newRand函数的并发安全性
func TestNewRandConcurrency(t *testing.T) {
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	randFn := newRand(keys)

	// 并发测试
	const numGoroutines = 100
	const numCallsPerGoroutine = 1000

	var wg sync.WaitGroup
	results := make([]string, 0, numGoroutines*numCallsPerGoroutine)
	var mu sync.Mutex

	// 启动多个goroutine并发调用
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numCallsPerGoroutine; j++ {
				result := randFn()
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// 验证结果
	if len(results) != numGoroutines*numCallsPerGoroutine {
		t.Errorf("期望 %d 个结果，实际得到 %d 个", numGoroutines*numCallsPerGoroutine, len(results))
	}

	// 验证所有结果都是有效的key
	for i, result := range results {
		valid := false
		for _, key := range keys {
			if result == key {
				valid = true
				break
			}
		}
		if !valid && result != "" {
			t.Errorf("结果 %d 包含无效key: %s", i, result)
		}
	}

	t.Logf("并发测试完成，总调用次数: %d", len(results))
}

// BenchmarkNewRand 性能基准测试
func BenchmarkNewRand(b *testing.B) {
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	randFn := newRand(keys)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = randFn()
		}
	})
}

// TestEmptyKeysConcurrency 测试空keys的并发安全性
func TestEmptyKeysConcurrency(t *testing.T) {
	randFn := newRand([]string{})

	const numGoroutines = 50
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				result := randFn()
				if result != "" {
					t.Errorf("空keys应该返回空字符串，但得到: %s", result)
				}
			}
		}()
	}

	wg.Wait()
}
