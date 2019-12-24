package main

type stringSlice []string

func (ss *stringSlice) String() string {
	return "String slice flag."
}

func (ss *stringSlice) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}
