package breaker

type State int

const (
	CLOSED State = iota
	OPEN
	HALF_OPEN
)

func (s State) String() string {
	switch s {
	case CLOSED:
		return "CLOSED"
	case OPEN:
		return "OPEN"
	case HALF_OPEN:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}
