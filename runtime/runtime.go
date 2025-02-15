// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	ErrNoExist  = fmt.Errorf("failpoint: failpoint does not exist")
	ErrDisabled = fmt.Errorf("failpoint: failpoint is disabled")

	failpoints   map[string]*Failpoint
	failpointsMu sync.RWMutex
	envTerms     map[string]string
)

func init() {
	failpoints = make(map[string]*Failpoint)
	envTerms = make(map[string]string)
	if s := os.Getenv("GOFAIL_FAILPOINTS"); len(s) > 0 {
		if fpMap, err := parseFailpoints(s); err != nil {
			fmt.Printf("fail to parse failpoint: %v\n", err)
			os.Exit(1)
		} else {
			envTerms = fpMap
		}
	}
	if s := os.Getenv("GOFAIL_HTTP"); len(s) > 0 {
		if err := serve(s); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func parseFailpoints(fps string) (map[string]string, error) {
	// The format is <FAILPOINT>=<TERMS>[;<FAILPOINT>=<TERMS>]*
	fpMap := map[string]string{}

	for _, fp := range strings.Split(fps, ";") {
		if len(fp) == 0 {
			continue
		}
		fpTerm := strings.Split(fp, "=")
		if len(fpTerm) != 2 {
			err := fmt.Errorf("bad failpoint %q", fp)
			return nil, err
		}
		fpMap[fpTerm[0]] = fpTerm[1]
	}
	return fpMap, nil
}

// Enable sets a failpoint to a given failpoint description.
func Enable(failpath, inTerms string) error {
	failpointsMu.Lock()
	defer failpointsMu.Unlock()
	return enable(failpath, inTerms)
}

// enable enables a failpoint
func enable(failpath, inTerms string) error {
	fp := failpoints[failpath]
	if fp == nil {
		return ErrNoExist
	}

	t, err := newTerms(failpath, inTerms)
	if err != nil {
		fmt.Printf("failed to enable \"%s=%s\" (%v)\n", failpath, inTerms, err)
		return err
	}
	fp.t = t

	return nil
}

// Disable stops a failpoint from firing.
func Disable(failpath string) error {
	failpointsMu.Lock()
	defer failpointsMu.Unlock()
	return disable(failpath)
}

func disable(failpath string) error {
	fp := failpoints[failpath]
	if fp == nil {
		return ErrNoExist
	}

	if fp.t == nil {
		return ErrDisabled
	}
	fp.t = nil

	return nil
}

// Status gives the current setting for the failpoint
func Status(failpath string) (string, error) {
	failpointsMu.Lock()
	defer failpointsMu.Unlock()
	return status(failpath)
}

func status(failpath string) (string, error) {
	fp := failpoints[failpath]
	if fp == nil {
		return "", ErrNoExist
	}

	t := fp.t
	if t == nil {
		return "", ErrDisabled
	}

	return t.desc, nil
}

func List() []string {
	failpointsMu.Lock()
	defer failpointsMu.Unlock()
	return list()
}

func list() []string {
	ret := make([]string, 0, len(failpoints))
	for fp := range failpoints {
		ret = append(ret, fp)
	}
	return ret
}

func register(failpath string, fp *Failpoint) {
	failpointsMu.Lock()
	failpoints[failpath] = fp
	failpointsMu.Unlock()
	if t, ok := envTerms[failpath]; ok {
		Enable(failpath, t)
	}
}
