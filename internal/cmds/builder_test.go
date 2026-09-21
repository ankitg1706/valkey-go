package cmds

import (
	"reflect"
	"testing"
)

func Benchmark_Builder_Get(b *testing.B) {
	builder := NewBuilder(NoSlot)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := builder.Get().Key("bench_key").Build()
		PutCompleted(cmd)
	}
}

func Benchmark_Builder_Set(b *testing.B) {
	builder := NewBuilder(NoSlot)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := builder.Set().Key("bench_key").Value("bench_value").Build()
		PutCompleted(cmd)
	}
}

func Benchmark_Builder_MGet(b *testing.B) {
	builder := NewBuilder(NoSlot)
	keys := []string{"k1", "k2", "k3", "k4"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := builder.Mget().Key(keys...).Build()
		PutCompleted(cmd)
	}
}

func Benchmark_Builder_HGetAll(b *testing.B) {
	builder := NewBuilder(NoSlot)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := builder.Hgetall().Key("bench_hash").Build()
		PutCompleted(cmd)
	}
}

func Benchmark_Builder_ComplexArgs(b *testing.B) {
	builder := NewBuilder(NoSlot)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := builder.Zadd().Key("bench_zset").Nx().Ch().ScoreMember().ScoreMember(1.5, "member1").ScoreMember(2.5, "member2").Build()
		PutCompleted(cmd)
	}
}

func TestPutCompleted(t *testing.T) {
retry:
	cs1 := get()
	cs1.s = append(cs1.s, "1", "1", "1", "1", "1")
	PutCompleted(Completed{cs: cs1})
	cs2 := get()
	if cs1 != cs2 {
		goto retry
	}
	if len(cs2.s) != 0 {
		t.Fatalf("Put doesn't clean the CommandSlice")
	}
}

func TestPutCompletedForce(t *testing.T) {
retry:
	cs1 := get()
	cs1.s = append(cs1.s, "1", "1", "1", "1", "1")
	cs1.r = 1 // pin
	PutCompletedForce(Completed{cs: cs1})
	cs2 := get()
	if cs1 != cs2 {
		goto retry
	}
	if len(cs2.s) != 0 {
		t.Fatalf("PutCompletedForce doesn't clean the CommandSlice")
	}
}

func TestPutCacheableForce(t *testing.T) {
retry:
	cs1 := get()
	cs1.s = append(cs1.s, "1", "1", "1", "1", "1")
	cs1.r = 1 // pin
	PutCacheableForce(Cacheable{cs: cs1})
	cs2 := get()
	if cs1 != cs2 {
		goto retry
	}
	if len(cs2.s) != 0 {
		t.Fatalf("PutCacheableForce doesn't clean the CommandSlice")
	}
}

func TestPutCacheable(t *testing.T) {
retry:
	cs1 := get()
	cs1.s = append(cs1.s, "1", "1", "1", "1", "1")
	PutCacheable(Cacheable{cs: cs1})
	cs2 := get()
	if cs1 != cs2 {
		goto retry
	}
	if len(cs2.s) != 0 {
		t.Fatalf("Put doesn't clean the CommandSlice")
	}
}

func TestArbitraryIsZero(t *testing.T) {
	builder := NewBuilder(NoSlot)
	if cmd := builder.Arbitrary("any", "cmd"); cmd.IsZero() {
		t.Fatalf("arbitrary failed")
	}
	var cmd Arbitrary
	if !cmd.IsZero() {
		t.Fatalf("arbitrary failed")
	}
}

func TestArbitrary(t *testing.T) {
	builder := NewBuilder(NoSlot)
	cmd := builder.Arbitrary("any", "cmd").Keys("k1", "k2").Args("a1", "a2")
	if c := cmd.Build(); !reflect.DeepEqual(c.Commands(), []string{"any", "cmd", "k1", "k2", "a1", "a2"}) {
		t.Fatalf("arbitrary failed")
	}
	if c := builder.Arbitrary("any").Blocking(); !c.IsBlock() {
		t.Fatalf("arbitrary failed")
	}
	if c := builder.Arbitrary("any").ReadOnly(); !c.IsReadOnly() {
		t.Fatalf("arbitrary failed")
	}

	builder2 := NewBuilder(InitSlot)

	defer func() {
		if e := recover(); e != multiKeySlotErr {
			t.Errorf("arbitrary not check slots")
		}
	}()

	builder2.Arbitrary().Keys("k1", "k2")
}

func TestEmptyArbitrary(t *testing.T) {
	builder := NewBuilder(NoSlot)
	defer func() {
		if e := recover(); e != arbitraryNoCommand {
			t.Errorf("arbitrary not check empty")
		}
	}()
	builder.Arbitrary().Build()
}

func TestEmptySubscribe(t *testing.T) {
	builder := NewBuilder(NoSlot)
	defer func() {
		if e := recover(); e != arbitrarySubscribe {
			t.Errorf("arbitrary not check subscribe command")
		}
	}()
	builder.Arbitrary("SUBSCRIBE").Build()
}

func TestEmptyArbitraryMultiGet(t *testing.T) {
	builder := NewBuilder(NoSlot)
	defer func() {
		if e := recover(); e != arbitraryNoCommand {
			t.Errorf("arbitrary not check empty")
		}
	}()
	builder.Arbitrary().MultiGet()
}

func TestArbitraryMultiGet(t *testing.T) {
	builder := NewBuilder(NoSlot)
	cacheable := Cacheable(builder.Arbitrary("MGET").Args("KKK").MultiGet())
	if !cacheable.IsMGet() {
		t.Fatalf("arbitrary failed")
	}
}

func TestArbitraryMultiGetPanic(t *testing.T) {
	builder := NewBuilder(NoSlot)
	defer func() {
		if e := recover(); e != arbitraryMultiGet {
			t.Errorf("arbitrary not check MGET command")
		}
	}()
	builder.Arbitrary("SUBSCRIBE").MultiGet()
}

func TestBuiltTwice(t *testing.T) {
	src := NewBuilder(NoSlot).Get()
	cmd1 := src.Key("a")
	cmd2 := src.Key("b")
	cmd1.Build()
	defer func() {
		if e := recover(); e != ErrBuiltTwice {
			t.Errorf("arbitrary not check MGET command")
		}
	}()
	cmd2.Build()
}

func TestVerify(t *testing.T) {
	src := NewBuilder(NoSlot).Get()
	cmd1 := src.Key("a").Build()
	cmd1.cs.Verify()
	src.Key("b")
	defer func() {
		if e := recover(); e != ErrUnfinished {
			t.Errorf("arbitrary not check MGET command")
		}
	}()
	cmd1.cs.Verify()
}
