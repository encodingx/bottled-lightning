package bottledlightning

import (
	"fmt"
)

func errorf(prefix string, errPtr *error) {
	if *errPtr == nil {
		return
	}

	*errPtr = fmt.Errorf("%s: %w", prefix, *errPtr)

	return
}
