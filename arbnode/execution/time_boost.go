package execution

import (
	"time"
)

// PriorityBid is a transaction that has a priority fee and a timestamp.
type PriorityBid interface {
	PriorityFee() uint64
	Timestamp() time.Time
	UpdateTimestamp(time.Time)
	Boosted() bool
}

// TimeBoostable represents a group of transactions that can be
// sorted by time boost score.
type TimeBoostable[T PriorityBid] struct {
	txs         []T
	gFactor     uint64
	constFactor uint64
}

type Opt[T PriorityBid] func(tb *TimeBoostable[T])

// WithMaxBoostFactor sets the maximum time boost factor, g.
func WithMaxBoostFactor[T PriorityBid](gFactor uint64) Opt[T] {
	return func(tb *TimeBoostable[T]) {
		tb.gFactor = gFactor
	}
}

// WithDenominatorConstant sets the constant factor in the time boost
// denominator calculation.
func WithDenominatorConstant[T PriorityBid](cFactor uint64) Opt[T] {
	return func(tb *TimeBoostable[T]) {
		tb.constFactor = cFactor
	}
}

// NewTimeBoostable creates an instance of time boostable transactions
// with optional parameters, setting defaults if not provided.
func NewTimeBoostable[T PriorityBid](txs []T, opts ...Opt[T]) *TimeBoostable[T] {
	tb := &TimeBoostable[T]{
		txs:         txs,
		gFactor:     500,
		constFactor: 50, // TODO: Decide
	}
	for _, o := range opts {
		o(tb)
	}
	return tb
}

func (tb *TimeBoostable[T]) Len() int      { return len(tb.txs) }
func (tb *TimeBoostable[T]) Swap(i, j int) { tb.txs[i], tb.txs[j] = tb.txs[j], tb.txs[i] }
func (tb *TimeBoostable[T]) Less(i, j int) bool {
	a := tb.txs[i]
	t1 := a.Timestamp().UnixMilli()
	if tb.canBoost(a) {
		delta := tb.computeBoostDelta(a.PriorityFee())
		t1 = int64(saturatingSub(t1, delta))
		a.UpdateTimestamp(time.UnixMilli(t1))
	}
	b := tb.txs[j]
	t2 := b.Timestamp().UnixMilli()
	if tb.canBoost(b) {
		delta := tb.computeBoostDelta(b.PriorityFee())
		t2 = int64(saturatingSub(t2, delta))
		b.UpdateTimestamp(time.UnixMilli(t2))
	}
	return t1 < t2
}

// Computes the boost delta for a given priority fee using the
// parameters of the timeboostable instance.
func (tb *TimeBoostable[T]) computeBoostDelta(prioFee uint64) int64 {
	return int64((prioFee * tb.gFactor) / (prioFee + tb.constFactor))
}

// Checks if a transaction can be time boosted.
func (tb *TimeBoostable[T]) canBoost(tx T) bool {
	return !tx.Boosted() && tx.PriorityFee() != 0
}

func saturatingSub(a, b int64) uint64 {
	if a < b {
		return 0
	}
	return uint64(a - b)
}
