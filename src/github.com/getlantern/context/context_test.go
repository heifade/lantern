package context

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStack(t *testing.T) {
	c := Enter()
	c.Put("a", 1)
	penultimate := Enter().
		Put("b", 2)
	c = Enter().
		PutDynamic("c", func() interface{} { return 4 }).
		Put("d", 5)

	// Put something in the penultimate context and make sure it doesn't override
	// what's set in the ultimate context
	penultimate.Put("c", 3)

	var assertMutex sync.Mutex
	assertContentsAsMap := func(expected Map) {
		m := AsMap()
		assertMutex.Lock()
		assert.Equal(t, expected, m)
		assertMutex.Unlock()
	}

	assertContentsAsRead := func(expected Map) {
		m := make(Map)
		Read(func(key string, value interface{}) {
			m[key] = value
		})
		assertMutex.Lock()
		assert.Equal(t, expected, m)
		assertMutex.Unlock()
	}

	assertContents := func(expected Map) {
		assertContentsAsMap(expected)
		assertContentsAsRead(expected)
	}

	assertContents(Map{
		"a": 1,
		"b": 2,
		"c": 4,
		"d": 5,
	})

	var wg sync.WaitGroup
	wg.Add(1)
	Go(func() {
		defer Enter().Put("e", 6).Exit()
		assertContents(Map{
			"a": 1,
			"b": 2,
			"c": 4,
			"d": 5,
			"e": 6,
		})
		wg.Done()
	})
	wg.Wait()

	wg.Add(1)
	Go(func() {
		// This goroutine doesn't Exit. Still, we shouldn't leak anything.
		wg.Done()
	})
	wg.Wait()

	assertContents(Map{
		"a": 1,
		"b": 2,
		"c": 4,
		"d": 5,
	})

	c = c.Exit()
	assert.NotNil(t, c)
	assertContents(Map{
		"a": 1,
		"b": 2,
		"c": 3,
	})

	c = c.Exit()
	assert.NotNil(t, c)
	assertContents(Map{
		"a": 1,
	})

	// Last exit
	assert.Nil(t, c.Exit())
	assertContents(Map{})

	// Exit again, just for good measure
	assert.Nil(t, c.Exit())
	assertContents(Map{})

	// Spawn a goroutine with no existing contexts
	wg.Add(1)
	Go(func() {
		defer Enter().Put("f", 7).Exit()
		assertContents(Map{
			"f": 7,
		})
		wg.Done()
	})
	wg.Wait()

	allmx.RLock()
	assert.Empty(t, contexts, "No contexts should be left")
	allmx.RUnlock()
}

func BenchmarkPut(b *testing.B) {
	c := Enter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Put("key", "value")
	}
}

func BenchmarkAsMapCheap(b *testing.B) {
	c := Enter().Put("a", 1).Put("b", 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.AsMap()
	}
}

func BenchmarkAsMapExpensive(b *testing.B) {
	Enter().Put("a", 1).Put("b", 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AsMap()
	}
}