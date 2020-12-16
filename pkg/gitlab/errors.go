package gitlab

type authError struct {
	error
}

func (*authError) Is(target error) bool {
	_, ok := target.(*authError)
	return ok
}
