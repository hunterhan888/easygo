package php

import (
	"syscall"
	"strings"
//	"strconv"
	"os"
	"bufio"
	"fmt"
)

func Explode(sep string, data string, n int) []string {
	return strings.SplitN(data, sep, n)
}

func FileExists(file string) bool {
	err := syscall.Access(file, syscall.O_RDONLY)
	return err == nil
}

func File_put_contents(outfile string, data string) error {
	file, err := os.Create(outfile)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writer.WriteString(data)
	writer.Flush()
	return err
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