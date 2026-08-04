package pool_test

import (
	"sync"
	"testing"

	"github.com/Den8319/shortener/pkg/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testItem — тестовая структура с методом Reset() для проверки пула.
type testItem struct {
	val  int
	str  string
	s    []int
	m    map[string]string
	strP *string
}

func (t *testItem) Reset() {
	if t == nil {
		return
	}
	t.val = 0
	t.str = ""
	t.s = t.s[:0]
	clear(t.m)
	if t.strP != nil {
		*t.strP = ""
	}
}

func TestNew(t *testing.T) {
	p := pool.New(func() *testItem {
		return &testItem{val: 42}
	})
	require.NotNil(t, p)

	item := p.Get()
	assert.Equal(t, 42, item.val, "new item should have factory-default value")
}

func TestPut_Get_ReusesObject(t *testing.T) {
	p := pool.New(func() *testItem {
		return &testItem{}
	})

	item := p.Get()
	item.val = 99
	item.str = "modified"
	item.s = append(item.s, 1, 2, 3)
	item.m = map[string]string{"key": "value"}

	p.Put(item)

	reused := p.Get()
	assert.Same(t, item, reused, "should reuse the same object from pool")
	assert.Equal(t, 0, reused.val, "val should be reset")
	assert.Equal(t, "", reused.str, "str should be reset")
	assert.Empty(t, reused.s, "slice should be reset")
	assert.Empty(t, reused.m, "map should be reset")
}

func TestPut_ResetsPointerField(t *testing.T) {
	p := pool.New(func() *testItem {
		return &testItem{}
	})

	strVal := "hello"
	item := p.Get()
	item.strP = &strVal

	p.Put(item)

	reused := p.Get()
	require.NotNil(t, reused.strP)
	assert.Equal(t, "", *reused.strP, "pointer field should be reset")
}

func TestGet_EmptyPool_CreatesNew(t *testing.T) {
	callCount := 0
	p := pool.New(func() *testItem {
		callCount++
		return &testItem{val: callCount}
	})

	item1 := p.Get()
	assert.Equal(t, 1, item1.val, "first Get should create new item")

	item2 := p.Get()
	assert.Equal(t, 2, item2.val, "second Get should create another new item")
	assert.NotSame(t, item1, item2, "items should be different instances")
}

func TestPool_Concurrent(t *testing.T) {
	p := pool.New(func() *testItem {
		return &testItem{}
	})

	var wg sync.WaitGroup
	workers := 100

	for range workers {
		wg.Go(func() {
			item := p.Get()
			item.val = 777
			item.str = "concurrent"
			p.Put(item)
		})
	}

	wg.Wait()

	// После всех операций пул должен содержать сброшенные объекты
	item := p.Get()
	assert.Equal(t, 0, item.val, "item from pool should be reset")
	assert.Equal(t, "", item.str, "item from pool should be reset")
}
