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
	priorityFee uint64
	timestamp   int64
}

func (m *mockTx) PriorityFee() uint64  { return m.priorityFee }
func (m *mockTx) Timestamp() time.Time { return time.UnixMilli(m.timestamp) }

func TestTimeBoost(t *testing.T) {
	t.Run("normalization", func(t *testing.T) {
		txes := []*mockTx{
			{priorityFee: 0, timestamp: 100},
			{priorityFee: 0, timestamp: 200},
			{priorityFee: 0, timestamp: 300},
			{priorityFee: 0, timestamp: 400},
		}
		tb := NewTimeBoost(txes)
		sort.Sort(tb)
		for i := 0; i < len(txes); i++ {
			if !txEq(txes[i], tb.txs[i]) {
				t.Fatalf("txes %d not equal: %v != %v", i, txes[i], tb.txs[i])
			}
		}
	})
	t.Run("boost simple case", func(t *testing.T) {

	})
	t.Run("not more boost than max g factor", func(t *testing.T) {

	})
	t.Run("constant factor effect", func(t *testing.T) {

	})
	t.Run("all boosted", func(t *testing.T) {

	})
	t.Run("intercalated", func(t *testing.T) {

	})
}

func TestTimeBoostable_computeBoostDelta(t *testing.T) {

}

func txEq(a, b PriorityBid) bool {
	return a.PriorityFee() == b.PriorityFee() && a.Timestamp() == b.Timestamp()
}
