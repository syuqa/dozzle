package docker_support

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

func contextDeadlineValue(ctx context.Context) string {
	if ctx == nil {
		return "none"
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return "none"
	}
	return time.Until(deadline).Round(time.Millisecond).String()
}

func diagnosticCaller(skip int) string {
	pcs := make([]uintptr, 12)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return "unknown"
	}

	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		name := frame.Function
		if name != "" &&
			!strings.Contains(name, "runtime.") &&
			!strings.Contains(name, "testing.") &&
			!strings.Contains(name, "stretchr/testify") {
			return fmt.Sprintf("%s:%d", shortFuncName(name), frame.Line)
		}
		if !more {
			break
		}
	}

	return "unknown"
}

func shortFuncName(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}
