package execution

import (
	"sort"
	"testing"
	"time"
)

var (
	_ sort.Interface = (*TimeBoostable[PriorityBid])(nil)
	_ PriorityBid    = (*txWithArrivalTime)(nil)
)

type mockTx struct {
	priorityFee    uint64
	timestamp      int64
	alreadyBoosted bool
}

func (m *mockTx) PriorityFee() uint64  { return m.priorityFee }
func (m *mockTx) Timestamp() time.Time { return time.UnixMilli(m.timestamp) }
func (m *mockTx) Boosted() bool        { return m.alreadyBoosted }
func (m *mockTx) UpdateTimestamp(tstamp time.Time) {
	if m.alreadyBoosted {
		return
	}
	m.timestamp = tstamp.UnixMilli()
	m.alreadyBoosted = true
}

func TestTimeBoost(t *testing.T) {
	t.Run("normalization = no bid no boost", func(t *testing.T) {
		want := []*mockTx{
			{priorityFee: 0, timestamp: 100},
			{priorityFee: 0, timestamp: 200},
			{priorityFee: 0, timestamp: 300},
			{priorityFee: 0, timestamp: 400},
		}
		tb := NewTimeBoostable(want)
		sort.Sort(tb)
		for i := 0; i < len(want); i++ {
			if !txEq(want[i], tb.txs[i]) {
				t.Fatalf("txes %d not equal: %v != %v", i, want[i], tb.txs[i])
			}
		}
	})
	t.Run("max bid", func(t *testing.T) {
		txes := []*mockTx{
			{priorityFee: 0, timestamp: 100},
			{priorityFee: 0, timestamp: 200},
			{priorityFee: 500, timestamp: 300},
		}
		want := []*mockTx{
			{priorityFee: 500, timestamp: 0},
			{priorityFee: 0, timestamp: 100},
			{priorityFee: 0, timestamp: 200},
		}
		tb := NewTimeBoostable(txes, WithMaxBoostFactor[*mockTx](500))
		sort.Sort(tb)
		for i := 0; i < len(want); i++ {
			if !txEq(want[i], tb.txs[i]) {
				t.Fatalf("txes %d not equal: %v != %v", i, want[i], tb.txs[i])
			}
		}
	})
	t.Run("bid just high enough to reorder", func(t *testing.T) {
		txes := []*mockTx{
			{priorityFee: 0, timestamp: 800},
			{priorityFee: 0, timestamp: 1000},
			{priorityFee: 100, timestamp: 1200},
		}
		want := []*mockTx{
			{priorityFee: 0, timestamp: 800},
			{priorityFee: 100, timestamp: 999},
			{priorityFee: 0, timestamp: 1000},
		}
		// A bid equal to the denominator constant of 100 would mean
		// the tx gets boosted by half the g factor.
		// (c * g) / (2c) = g / 2,
		// which in this case will be 402/2 = 201, just enough to beat out
		// the tx with timestamp 1000 by 1 millisecond.
		tb := NewTimeBoostable(
			txes,
			WithMaxBoostFactor[*mockTx](402),
			WithDenominatorConstant[*mockTx](100),
		)
		sort.Sort(tb)
		for i := 0; i < len(want); i++ {
			if !txEq(want[i], tb.txs[i]) {
				t.Fatalf("txes %d not equal: %v != %v", i, want[i], tb.txs[i])
			}
		}
	})
	t.Run("bid not high enough to reorder", func(t *testing.T) {
		txes := []*mockTx{
			{priorityFee: 0, timestamp: 800},
			{priorityFee: 0, timestamp: 1000},
			{priorityFee: 100, timestamp: 1200},
		}
		want := []*mockTx{
			{priorityFee: 0, timestamp: 800},
			{priorityFee: 0, timestamp: 1000},
			{priorityFee: 100, timestamp: 1001},
		}
		// A bid equal to the denominator constant of 100 would mean
		// the tx gets boosted by half the g factor.
		// (c * g) / (2c) = g / 2,
		// which in this case will be 398/2 = 199, which is not enough to beat
		// other txs by 1 millisecond.
		tb := NewTimeBoostable(
			txes,
			WithMaxBoostFactor[*mockTx](398),
			WithDenominatorConstant[*mockTx](100),
		)
		sort.Sort(tb)
		for i := 0; i < len(want); i++ {
			if !txEq(want[i], tb.txs[i]) {
				t.Fatalf("txes %d not equal: %v != %v", i, want[i], tb.txs[i])
			}
		}
	})
	t.Run("cannot boost more than max g factor", func(t *testing.T) {
		gFactor := uint64(500)
		txes := []*mockTx{
			{priorityFee: 0, timestamp: 800},
			{priorityFee: 0, timestamp: 1000},
			{priorityFee: gFactor * 10, timestamp: 1400},
		}
		// Even with a massive priority fee, the max boost factor will be g.
		want := []*mockTx{
			{priorityFee: 0, timestamp: 800},
			{priorityFee: gFactor * 10, timestamp: 1400 - int64(gFactor)},
			{priorityFee: 0, timestamp: 1000},
		}
		tb := NewTimeBoostable(
			txes,
			WithMaxBoostFactor[*mockTx](gFactor),
			WithDenominatorConstant[*mockTx](0),
		)
		sort.Sort(tb)
		for i := 0; i < len(want); i++ {
			if !txEq(want[i], tb.txs[i]) {
				t.Fatalf("txes %d not equal: %v != %v", i, want[i], tb.txs[i])
			}
		}
	})
	t.Run("all boosted with close timestamps results in ordering by bid", func(t *testing.T) {
		gFactor := uint64(500)
		cFactor := uint64(200)
		txes := []*mockTx{
			{priorityFee: 300, timestamp: 1000},
			{priorityFee: 200, timestamp: 1001},
			{priorityFee: 400, timestamp: 1002},
		}
		want := []*mockTx{
			{priorityFee: 400},
			{priorityFee: 300},
			{priorityFee: 200},
		}
		tb := NewTimeBoostable(
			txes,
			WithMaxBoostFactor[*mockTx](gFactor),
			WithDenominatorConstant[*mockTx](cFactor),
		)
		for i, tx := range txes {
			t.Logf("Tx %d got %d\n", i, tb.computeBoostDelta(tx.PriorityFee()))
		}
		sort.Sort(tb)
		for i := 0; i < len(want); i++ {
			a := want[i]
			b := tb.txs[i]
			if a.priorityFee != b.priorityFee {
				t.Fatalf("txes %d not equal: %v != %v", i, want[i], tb.txs[i])
			}
		}
	})
	t.Run("intercalated", func(t *testing.T) {
		txes := []*mockTx{
			{priorityFee: 0, timestamp: 700},
			{priorityFee: 100, timestamp: 1200},
			{priorityFee: 0, timestamp: 1001},
			{priorityFee: 400, timestamp: 1200},
			{priorityFee: 0, timestamp: 1002},
			{priorityFee: 200, timestamp: 1200},
			{priorityFee: 0, timestamp: 1000},
			{priorityFee: 200, timestamp: 1200},
		}
		want := []*mockTx{
			{priorityFee: 0, timestamp: 700},
			{priorityFee: 400, timestamp: 800},
			{priorityFee: 200, timestamp: 867},
			{priorityFee: 200, timestamp: 867},
			{priorityFee: 100, timestamp: 950},
			{priorityFee: 0, timestamp: 1000},
			{priorityFee: 0, timestamp: 1001},
			{priorityFee: 0, timestamp: 1002},
		}
		tb := NewTimeBoostable(
			txes,
			WithMaxBoostFactor[*mockTx](500),
			WithDenominatorConstant[*mockTx](100),
		)
		sort.Sort(tb)
		for i := 0; i < len(want); i++ {
			if !txEq(want[i], tb.txs[i]) {
				t.Fatalf("txes %d not equal: %v != %v", i, want[i], tb.txs[i])
			}
		}
	})
}

func TestTimeBoostable_computeBoostDelta(t *testing.T) {
	tests := []struct {
		name        string
		gFactor     uint64
		cFactor     uint64
		priorityFee uint64
		want        int64
	}{
		{
			name:        "0 priority fee",
			gFactor:     500,
			cFactor:     100,
			priorityFee: 0,
			want:        0,
		},
		{
			name:        "0 const factor gives g factor boost",
			gFactor:     500,
			cFactor:     0,
			priorityFee: 100,
			want:        500,
		},
		{
			name:        "const factor == g factor gives half g factor boost",
			gFactor:     500,
			cFactor:     100,
			priorityFee: 100,
			want:        250,
		},
		{
			name:        "approaches g factor but does not reach it (asymptote)",
			gFactor:     500,
			cFactor:     100,
			priorityFee: 100000,
			want:        499,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := &TimeBoostable[*mockTx]{
				gFactor:     tt.gFactor,
				constFactor: tt.cFactor,
			}
			if got := tb.computeBoostDelta(tt.priorityFee); got != tt.want {
				t.Errorf("computeBoostDelta() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkTimeBoost_10(b *testing.B) {
	benchBoost(b, 100)
}

func BenchmarkTimeBoost_100(b *testing.B) {
	benchBoost(b, 1000)
}

func BenchmarkTimeBoost_1000(b *testing.B) {
	benchBoost(b, 1000)
}

func benchBoost(b *testing.B, numTxs int) {
	b.Helper()
	b.StopTimer()
	txs := make([]*mockTx, numTxs)
	for i := 0; i < numTxs; i++ {
		txs[i] = &mockTx{
			priorityFee: uint64(i),
			timestamp:   int64(i),
		}
	}
	tb := NewTimeBoostable(
		txs,
		WithMaxBoostFactor[*mockTx](500),
		WithDenominatorConstant[*mockTx](100),
	)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		sort.Sort(tb)
	}
}

func TestTimeBoostable_canBoost(t *testing.T) {
	tests := []struct {
		name string
		tx   *mockTx
		want bool
	}{
		{
			name: "already boosted",
			tx:   &mockTx{alreadyBoosted: true},
			want: false,
		},
		{
			name: "no priority fee",
			tx:   &mockTx{alreadyBoosted: false, priorityFee: 0},
			want: false,
		},
		{
			name: "priority fee, not already boosted",
			tx:   &mockTx{alreadyBoosted: false, priorityFee: 1},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTimeBoostable(
				[]*mockTx{tt.tx},
			)
			if got := tb.canBoost(tt.tx); got != tt.want {
				t.Errorf("canBoost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_saturatingSub(t *testing.T) {
	type args struct {
		a int64
		b int64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "normal",
			args: args{a: 100, b: 50},
			want: 50,
		},
		{
			name: "negative",
			args: args{a: 50, b: 100},
			want: 0,
		},
		{
			name: "zero",
			args: args{a: 50, b: 50},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := saturatingSub(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("saturatingSub() = %v, want %v", got, tt.want)
			}
		})
	}
}

func txEq(a, b PriorityBid) bool {
	return a.PriorityFee() == b.PriorityFee() && a.Timestamp() == b.Timestamp()
}
