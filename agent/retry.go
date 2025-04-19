package agent

import (
	"time"

	"github.com/sohaha/zlsgo/zutil"
	"github.com/zlsgo/zllm/utils"
)

func doRetry(provider string, max int, fn func() (retry bool, err error)) (err error) {
	i := -1
	retry := false

	retryErr := zutil.DoRetry(max, func() error {
		i++
		if i > 0 {
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			utils.Warn(provider, "retrying", i, errMsg)
		}

		retry, err = fn()
		if !retry && err != nil {
			return nil
		}
		return err
	}, func(rc *zutil.RetryConf) {
		rc.BackOffDelay = true
		rc.MaxRetryInterval = time.Second * 8
	})
	if retryErr != nil {
		return retryErr
	}
	return err
}
