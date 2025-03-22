package agent

import (
	"github.com/sohaha/zlsgo/zarray"
)

func newRand(keys []string) func() string {
	k := make([]string, 0, len(keys))

	k = append(k, zarray.Shuffle(keys)...)

	return func() string {
		if len(k) == 1 {
			return k[0]
		}
		if len(k) == 0 {
			return ""
		}

		v := k[0]
		k = k[1:]
		return v
	}
}
