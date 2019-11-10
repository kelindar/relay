package relay

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testFile = "src/relay.lua"

func newRelay() *Relay {
	r, _ := New()
	f, _ := os.Open(testFile)
	r.vm.Update(f)
	return r
}

func TestGet(t *testing.T) {

	r, err := New()
	assert.NoError(t, err)

	f, err := os.Open(testFile)
	assert.NoError(t, err)

	err = r.vm.Update(f)
	assert.NoError(t, err)

	r.Get(context.Background())
	assert.Fail(t, "xxx")
}

func Benchmark_Get_Serial(b *testing.B) {
	r := newRelay()
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r.Get(context.Background())
	}
}

func Benchmark_Get_Parallel(b *testing.B) {
	r := newRelay()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.Get(context.Background())
		}
	})
}
