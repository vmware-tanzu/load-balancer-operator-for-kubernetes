// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type VIPRunner struct {
	WorkerEndpoint string
}

func NewVIPRunner(workerEndpoint string) *VIPRunner {
	return &VIPRunner{
		WorkerEndpoint: workerEndpoint,
	}
}

type nsipResp struct {
	IP      string `json:"ip"`
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway"`
}

func AllocVIP(runner *VIPRunner) (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:4827/nsips", runner.WorkerEndpoint))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	nresp := &nsipResp{}
	err = json.Unmarshal(body, nresp)
	if err != nil {
		return "", err
	}
	return nresp.IP, nil

}
