package php

import (
	"strings"
//	"strconv"
	"os"
	"fmt"
)

func Explode(sep string, data string, n int) []string {
	return strings.SplitN(data, sep, n)
}

func Uniqid() (string, error) {
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return "", err
	}
	defer f.Close()

	b := make([]byte, 16)
	_, err = f.Read(b)
	if err != nil {
		return "", err
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid, nil
}