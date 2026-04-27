package filesystem

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Policy struct {
	DeniedMap   []byte
	ExcludeList []byte
}

var (
	policyOnce sync.Once
	policyErr  error

	deniedMap struct {
		Dirs       []string `json:"dirs"`
		Files      []string `json:"files"`
		Prefixes   []string `json:"prefixes"`
		Extensions []string `json:"extensions"`
	}
	excludeList []string
)

func New(p Policy) error {
	policyOnce.Do(func() {
		if len(p.DeniedMap) > 0 {
			if err := json.Unmarshal(p.DeniedMap, &deniedMap); err != nil {
				policyErr = fmt.Errorf("DeniedMap unmarshal: %w", err)
				return
			}
		}
		if len(p.ExcludeList) > 0 {
			if err := json.Unmarshal(p.ExcludeList, &excludeList); err != nil {
				policyErr = fmt.Errorf("ExcludeList unmarshal: %w", err)
				return
			}
		}
	})
	return policyErr
}
