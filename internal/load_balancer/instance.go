package load_balancer

type Instance struct {
	ID                string
	URL               string
	Weight            int
	ActiveConnections int64
	Healthy           bool
}
