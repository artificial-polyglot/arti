package logger

import (
	"context"
	"fmt"
	"strconv"
	"testing"
)

func TestLogger_Main(t *testing.T) {
	Warn(context.Background(), 0, "Sample Error")
	//	Error(context.Background(), "Error Message")
}

func TestPanic(t *testing.T) {
	Panic(context.Background(), "Panic Message")
}

func TestFatal(t *testing.T) {
	ctx := context.Background()
	Fatal(ctx, "Error Message")
}

func TestDebug(t *testing.T) {
	ctx := context.Background()
	Debug(ctx, "Debug Message")
}

func TestError(t *testing.T) {
	var request = `
AudioData: 
  BibleBrain: 
    MP3_64: true`
	ctx := context.WithValue(context.Background(), "request", request)
	_, err := strconv.Atoi("12c")
	derr := Error(ctx, 500, err, "Error Message", 123, "part3", 34.5)
	fmt.Println(derr)
	fmt.Println(derr.Trace)
}
