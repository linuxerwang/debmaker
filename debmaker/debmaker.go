package debmaker

import "os"

// DebControl specifies the attributes of a deb control file.
type DebControl struct {
	PkgName    string            `cfg-attr:"pkg-name"`
	Maintainer string            `cfg-attr:"maintainer"`
	Desc       string            `cfg-attr:"description"`
	Attrs      map[string]string `cfg-attr:"other-attrs"`

	Arch    string
	Version string
}

// FileEntry specifies a input file and it's path in the debian package.
type FileEntry struct {
	Path    string `cfg-attr:"path"`     // input file path
	DebPath string `cfg-attr:"deb-path"` // path in deb file

	FileInfo os.FileInfo
	Md5sum   string
}

// Symlink specifies an absolute symbolic link for data files.
type Symlink struct {
	From string `cfg-attr:"from"`
	To   string `cfg-attr:"to"`
}

// DebSpec specifies the whole deb structure: control files and data files.
type DebSpec struct {
	DebCtrl *DebControl  `cfg-attr:"control"`
	Debian  []*FileEntry `cfg-attr:"debian"`
	Content []*FileEntry `cfg-attr:"content"`
	Link    []*Symlink   `cfg-attr:"link"`
}
