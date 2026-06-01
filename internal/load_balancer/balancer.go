package load_balancer

type Balancer interface {
	NextInstance() (*Instance, error)
}
