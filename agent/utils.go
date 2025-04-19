package agent

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

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

func isRetry(status int, msg string) (bool, error) {
	if status != http.StatusOK {
		errMsg := msg
		if errMsg == "" {
			errMsg = fmt.Sprintf("status code: %d", status)
		}
		if zarray.Contains([]int{http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusGatewayTimeout, http.StatusTooManyRequests}, status) {
			if status == http.StatusTooManyRequests && strings.Contains(errMsg, "quota") {
				return false, errors.New(errMsg)
			}
			return true, errors.New(errMsg)
		}
		return false, errors.New(errMsg)
	}
	return false, nil
}
