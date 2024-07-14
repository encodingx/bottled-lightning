package bottledlightning

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorf(t *testing.T) {
	var (
		e error

		f = func(err bool) (erred bool, e error) {
			defer errorf("oops", &e)

			if err {
				e = errors.New("ka-BOOM!")
			}

			return
		}
	)

	_, e = f(false)

	assert.Nil(t, e)

	_, e = f(true)

	assert.Equal(t, "oops: ka-BOOM!",
		e.Error(),
	)

	return
}
