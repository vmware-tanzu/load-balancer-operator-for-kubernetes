// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
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
