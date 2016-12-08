package system

type FakeFreeTcpPort struct {
	FreeTcpPort
	Cb func()
}

func (f *FakeFreeTcpPort) FindFreePortInRange(minport int, maxport int) (int, error) {
	f.Cb()
	return 8080, nil
}