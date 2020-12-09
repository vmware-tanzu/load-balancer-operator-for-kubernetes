// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"math/rand"
	"time"
)

const (
	numerics   = "0123456789"
	specials   = "~=+%^*/()[]{}/!@#$?|"
	uppercases = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercases = "abcdefghijklmnopqrstuvwxyz"
)

// GenereatePassword generate a random password
func GenereatePassword(length int, mustHaveLowercase, mustHaveUppercase, mustHaveSpecial, mustHaveNumeric bool) string {
	rand.Seed(time.Now().UnixNano())
	all := uppercases + lowercases + numerics + specials
	buf := make([]byte, length)
	i := 0

	if mustHaveLowercase {
		buf[i] = lowercases[rand.Intn(len(lowercases))]
		i++
	}

	if mustHaveUppercase {
		buf[i] = uppercases[rand.Intn(len(uppercases))]
		i++
	}

	if mustHaveSpecial {
		buf[i] = specials[rand.Intn(len(specials))]
		i++
	}

	if mustHaveNumeric {
		buf[i] = numerics[rand.Intn(len(numerics))]
		i++
	}

	for ; i < length; i++ {
		buf[i] = all[rand.Intn(len(all))]
	}

	rand.Shuffle(len(buf), func(i, j int) {
		buf[i], buf[j] = buf[j], buf[i]
	})

	return string(buf)
}
