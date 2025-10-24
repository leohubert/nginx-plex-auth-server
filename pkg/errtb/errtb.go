package errtb

func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func Must0(err error) {
	if err != nil {
		panic(err)
	}
}
