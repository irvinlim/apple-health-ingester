package testutils

import (
	"github.com/stretchr/testify/assert"
)

// WantError checks err against assert.ErrorAssertionFunc, returning true if an
// error was encountered for short-circuiting.
func WantError(t assert.TestingT, wantErr assert.ErrorAssertionFunc, err error, i ...interface{}) bool {
	if wantErr == nil {
		wantErr = assert.NoError
	}
	wantErr(t, err, i...)
	return err != nil
}

// AssertErrorContains returns assert.ErrorAssertionFunc that asserts that the error
// message contains str.
func AssertErrorContains(str string) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		return assert.Contains(t, err.Error(), str, i...)
	}
}
