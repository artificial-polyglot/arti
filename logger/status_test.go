package logger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestStatus(t *testing.T) {
	err := errors.New("A go error")
	fmt.Println("1. ===")
	status := Error(context.Background(), 500, err, "Arti Error")
	fmt.Println("2. ===")
	fmt.Println(status)
	fmt.Println("3. ===")
	data, err := json.MarshalIndent(status, "", "  ")
	fmt.Println(string(data))

}
