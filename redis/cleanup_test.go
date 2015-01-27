package redis_test

type CleanupFunc func()
type CleanupFuncs []CleanupFunc

func (fns CleanupFuncs) Cleanup() {
	for i := len(fns) - 1; i >= 0; i-- {
		fns[i]()
	}
}

func (fns *CleanupFuncs) Register(fn CleanupFunc) {
	(*fns) = append(*fns, fn)
}
