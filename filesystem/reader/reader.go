package reader

type ListOption struct {
	SkipExcluded      bool
	IgnoreWalkError   bool
	IncludeNonRegular bool
}

func getListOption(opts []ListOption) ListOption {
	if len(opts) == 0 {
		return ListOption{}
	}
	return opts[len(opts)-1]
}
