package cli

func SetInteractiveStdin(f func() bool) func() {
	old := isInteractiveStdin
	isInteractiveStdin = f
	return func() { isInteractiveStdin = old }
}
