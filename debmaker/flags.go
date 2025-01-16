package debmaker

type Params struct {
	OutputDir string
	Spec      string
	Desc      string
	Version   string
	Arch      string
	Verbose   bool
	PkgName   string
	Postinst  string
	Prerm     string
	TmpDir    string

	Files stringSlice
	Dirs  stringSlice
}

type stringSlice []string

func (ss *stringSlice) String() string {
	return "String slice flag."
}

func (ss *stringSlice) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}
