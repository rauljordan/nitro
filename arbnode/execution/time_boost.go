package execution

import (
	"time"
)

// PriorityBid is a transaction that has a priority fee and a timestamp.
type PriorityBid interface {
	PriorityFee() uint64
	Timestamp() time.Time
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

// NewTimeBoost creates an instance of time boostable transactions
// with optional parameters, setting defaults if not provided.
func NewTimeBoost[T PriorityBid](txs []T, opts ...Opt[T]) *TimeBoostable[T] {
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
	if a.PriorityFee() != 0 && a.PriorityFee() <= tb.gFactor {
		delta := tb.computeBoostDelta(a.PriorityFee())
		t1 -= int64(delta)
	}
	b := tb.txs[j]
	t2 := b.Timestamp().UnixMilli()
	if b.PriorityFee() != 0 && b.PriorityFee() <= tb.gFactor {
		delta := tb.computeBoostDelta(b.PriorityFee())
		t2 -= int64(delta)
	}
	return t1 < t2
}

// Computes the boost delta for a given priority fee using the
// parameters of the timeboostable instance.
func (tb *TimeBoostable[T]) computeBoostDelta(prioFee uint64) int64 {
	return int64((prioFee * tb.gFactor) / (prioFee + tb.constFactor))
}
