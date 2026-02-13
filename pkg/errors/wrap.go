package errutils

import (
	"fmt"
	"path"
	"runtime"
)

func WrapPathErr(err error) error {
	if err == nil {
		return nil
	}

	pc, _, line, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("[unknown:0] %w", err)
	}

	fullFn := runtime.FuncForPC(pc).Name()
	fnName := path.Base(fullFn)
	return fmt.Errorf("[%s:%d] %w", fnName, line, err)
}
