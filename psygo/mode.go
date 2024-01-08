package psygo

import "github.com/Psychopath-H/psyweb-master/psygo/binding"

const (
	// DebugMode indicates psygo mode is debug.
	DebugMode = "debug"
	// ReleaseMode indicates psygo mode is release.
	ReleaseMode = "release"
	// TestMode indicates psygo mode is test.
	TestMode = "test"
)

const (
	debugCode = iota
	releaseCode
	testCode
)

func EnableJsonDecoderDisallowUnknownFields() {
	binding.DisallowUnknownFields = true
}

func DisableLocalBindValidation() {
	binding.UsingLocalValidate = false
}
