package testutil

type FatalFormatter interface {
	Fatalf(string, ...interface{})
}
