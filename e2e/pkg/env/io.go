// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"io/ioutil"
	"os"
)

func NotExist(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func ReadFromFile(file string) ([]byte, error) {
	data, err := ioutil.ReadFile(file)
	return data, err
}
