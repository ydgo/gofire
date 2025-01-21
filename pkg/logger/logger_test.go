package logger

import (
	"testing"
	"time"
)

func TestA(t *testing.T) {
	t.Log(time.Duration(15000000000).Seconds())
}
