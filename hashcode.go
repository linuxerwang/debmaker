package main

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var hash = md5.New()

type dirCache struct {
	cache map[string]struct{}
}

func NewDirCache() *dirCache {
	return &dirCache{
		cache: make(map[string]struct{}),
	}
}

func (dc *dirCache) parse(path string) []string {
	dirs := []string{}
	for d := path; d != "/" && d != "."; d = filepath.Dir(d) {
		if _, ok := dc.cache[d]; ok {
			break
		}

		dc.cache[d] = struct{}{}
		dirs = append(dirs, d)
	}
	return dirs
}

type dirFileInfo struct {
	name string
}

func (dfi dirFileInfo) Name() string {
	return dfi.name
}

func (dfi dirFileInfo) Size() int64 {
	return 0
}

func (dfi dirFileInfo) Mode() os.FileMode {
	return 0755
}

func (dfi dirFileInfo) ModTime() time.Time {
	return time.Now()
}

func (dfi dirFileInfo) IsDir() bool {
	return true
}

func (dfi dirFileInfo) Sys() interface{} {
	return nil
}

func NewDirFileInfo(name string) *dirFileInfo {
	return &dirFileInfo{
		name: name,
	}
}

// fillFileInfo fills in the FileInfo field of each content file entry and returns the new file slice.
// For directories, it recursively walk the tree to collect each directory or file.
func fillFileInfo(contentFiles []*FileEntry) ([]*FileEntry, error) {
	newDirs := []*FileEntry{}
	newFiles := []*FileEntry{}
	newSymlinks := []*FileEntry{}

	pdirs := NewDirCache()

	for _, f := range contentFiles {
		fi, err := os.Lstat(f.Path)
		if err != nil {
			return nil, err
		}

		if fi.IsDir() {
			err := filepath.Walk(f.Path, func(path string, info os.FileInfo, err error) error {
				if info.IsDir() {
					relpath, err := filepath.Rel(f.Path, path)
					if err != nil {
						if *verbose {
							fmt.Printf("Failed to get relative path for %s against %s.", path, f.Path)
						}
						return err
					}

					de := &FileEntry{
						Path:     path,
						DebPath:  filepath.Join(f.DebPath, relpath),
						FileInfo: info,
					}

					if _, ok := pdirs.cache[de.DebPath]; !ok {
						newDirs = append(newDirs, de)
					}
					return nil
				}

				relpath, err := filepath.Rel(f.Path, path)
				if err != nil {
					if *verbose {
						fmt.Printf("Failed to get relative path for %s against %s.", path, f.Path)
					}
					return err
				}

				fe := &FileEntry{
					Path:     path,
					DebPath:  filepath.Join(f.DebPath, relpath),
					FileInfo: info,
				}

				if err = fillMd5Sum(fe); err != nil {
					return err
				}

				if info.Mode()&os.ModeSymlink != 0 {
					newSymlinks = append(newSymlinks, fe)
				} else {
					newFiles = append(newFiles, fe)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else if fi.Mode()&os.ModeSymlink != 0 {
			f.FileInfo = fi
			newSymlinks = append(newSymlinks, f)
		} else {
			f.FileInfo = fi
			if err = fillMd5Sum(f); err != nil {
				return nil, err
			}

			newFiles = append(newFiles, f)

			// Make sure the file's directory exists.
			for _, d := range pdirs.parse(filepath.Dir(f.DebPath)) {
				de := &FileEntry{
					Path:     "",
					DebPath:  d,
					FileInfo: NewDirFileInfo(d),
				}

				newDirs = append(newDirs, de)
			}
		}
	}

	sort.Slice(newDirs, func(i, j int) bool {
		return newDirs[i].DebPath < newDirs[j].DebPath
	})
	return append(append(newDirs, newFiles...), newSymlinks...), nil
}

// fillMd5Sum fills the md5sum of the given content file entry.
func fillMd5Sum(cf *FileEntry) error {
	var md5sum string
	var err error
	if cf.FileInfo.Mode()&os.ModeSymlink != os.ModeSymlink {
		if md5sum, err = calculateMd5sum(cf.Path, cf.FileInfo.Size()); err != nil {
			return err
		}
		cf.Md5sum = md5sum
	}

	return nil
}

// calcMd5Sum calculates the md5sum of the given file.
func calculateMd5sum(path string, size int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash.Reset()
	buffer := make([]byte, 16384)
	var num int
	var n int64
	for n = 0; n < size; {
		num, err = f.Read(buffer)
		if err != nil {
			if *verbose {
				fmt.Printf("Failed to read from file %s.", path, path)
			}
			return "", err
		}

		n += int64(num)
		hash.Write(buffer[:num])
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
