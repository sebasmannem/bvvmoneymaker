package internal

import (
	"encoding/json"
	"fmt"
)

// Use this function to print a human readable version of the returned struct.
func PrettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return
}

func DefaultInt64(value int64, def int64) int64 {
	if value == 0 {
		return def
	} else {
		return value
	}
}