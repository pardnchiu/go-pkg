package reader

import (
	"os"

	"github.com/pardnchiu/go-pkg/utils"
)

type ListOption struct {
	SkipExcluded      bool
	SkipDenied        bool
	IgnoreWalkError   bool
	IncludeNonRegular bool
	Context           int
}

func getListOption(opts []ListOption) ListOption {
	if len(opts) == 0 {
		return ListOption{}
	}
	return opts[len(opts)-1]
}

type File struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
	Matches []Line `json:"matches,omitempty"`
}

type Line struct {
	Line    int    `json:"line"`
	Text    string `json:"text"`
	Context bool   `json:"context,omitempty"`
}

func newFile(path string, info os.FileInfo) File {
	return File{
		Name:    info.Name(),
		Path:    utils.AbsPath("", path),
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime().Format("2006-01-02 15:04"),
	}
}
