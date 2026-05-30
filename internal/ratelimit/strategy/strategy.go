package strategy

type Strategy interface {
	Allow() bool
}

var (
	_ Strategy = (*TokenBucket)(nil)
	_ Strategy = (*FixedWindow)(nil)
	_ Strategy = (*SlidingCounter)(nil)
	_ Strategy = (*SlidingLog)(nil)
)
