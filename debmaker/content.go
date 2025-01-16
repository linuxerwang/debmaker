package debmaker

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/linuxerwang/ar"
)

const (
	isLink  = 0120000 // Symbolic link
	dataTgz = "data.tar.gz"
)

func AddContentFiles(arw *ar.Writer, debSpec *DebSpec, tmpDir string, verbose bool) error {
	if err := createDataTgz(debSpec, tmpDir, verbose); err != nil {
		return err
	}

	tmpData := filepath.Join(tmpDir, dataTgz)

	info, err := os.Stat(tmpData)
	if err != nil {
		return err
	}

	f, err := os.Open(tmpData)
	if err != nil {
		return err
	}
	defer f.Close()

	hdr := &ar.Header{
		Size:    info.Size(),
		Name:    dataTgz,
		ModTime: time.Now(),
		Mode:    0777,
	}
	if err := arw.WriteHeader(hdr); err != nil {
		if verbose {
			fmt.Printf("Failed to write header for %s.\n", dataTgz)
		}
		return err
	}
	if _, err := io.Copy(arw, f); err != nil {
		if verbose {
			fmt.Printf("Failed to copy %s to deb file.\n", dataTgz)
		}
		return err
	}

	fmt.Printf("Added %s to deb file.\n", dataTgz)

	return nil
}

func createDataTgz(debSpec *DebSpec, tmpDir string, verbose bool) error {
	tmpData := filepath.Join(tmpDir, dataTgz)
	f, err := os.Create(tmpData)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	w := tar.NewWriter(gz)
	defer w.Close()

	// Write data files
	for _, df := range debSpec.Content {
		info := df.FileInfo

		hdr := &tar.Header{
			Name:    df.DebPath,
			Mode:    0644,
			ModTime: info.ModTime(),
		}
		perm := info.Mode().Perm()
		if perm&0001 > 0 {
			hdr.Mode |= 0001
		}
		if perm&0010 > 0 {
			hdr.Mode |= 0010
		}
		if perm&0100 > 0 {
			hdr.Mode |= 0100
		}
		if info.IsDir() {
			// Handle directory
			hdr.Size = 0
			hdr.Typeflag |= tar.TypeDir

			if !strings.HasSuffix(hdr.Name, "/") {
				hdr.Name += "/"
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			// Handle symlink
			hdr.Size = 0
			hdr.Typeflag |= tar.TypeSymlink
			hdr.Mode |= isLink

			l, err := os.Open(df.Path)
			if err != nil {
				fmt.Printf("Failed to open symlink %s, %v.\n", df.Path, err)
				os.Exit(1)
			}
			b, err := ioutil.ReadAll(l)
			l.Close()
			if err != nil {
				fmt.Printf("Failed to close symlink %s, %v.\n", df.Path, err)
				os.Exit(1)
			}

			hdr.Linkname = strings.TrimSpace(string(b))
		} else {
			hdr.Size = info.Size()
		}

		if err := w.WriteHeader(hdr); err != nil {
			if verbose {
				fmt.Printf("Failed to write header for file %s.\n", info.Name())
			}
			return err
		}
		if info.IsDir() {
			if verbose {
				fmt.Printf("Added directory %s to %s.\n", df.DebPath, dataTgz)
			}
			continue
		} else if info.Mode()&os.ModeSymlink != 0 {
			if verbose {
				fmt.Printf("Added symlink %s to %s.\n", df.DebPath, dataTgz)
			}
			continue
		}

		// Note that the opened file has to be closed explicitly
		in, err := os.Open(df.Path)
		if err != nil {
			if verbose {
				fmt.Printf("Failed to open file %s.\n", df.Path)
			}
			return err
		}

		_, err = io.Copy(w, in)
		if err != nil {
			if verbose {
				fmt.Printf("Failed to copy content from %s to %s.\n", df.Path, dataTgz)
			}
			in.Close()
			return err
		}

		if err = in.Close(); err != nil {
			if verbose {
				fmt.Printf("Failed to close file %s.\n", df.Path)
			}
			return err
		}

		if verbose {
			fmt.Printf("Added file %s to %s.\n", df.DebPath, dataTgz)
		}
	}

	// Write symlinks
	for _, l := range debSpec.Link {
		hdr := &tar.Header{
			Typeflag: tar.TypeSymlink,
			Name:     l.To,
			Mode:     0755 | isLink,
			ModTime:  time.Now(),
			Linkname: l.From,
			Size:     0,
		}
		if err := w.WriteHeader(hdr); err != nil {
			if verbose {
				fmt.Printf("Failed to write header for symlink %s.\n", l.To)
			}
			return err
		}
	}

	fmt.Printf("Created temporary file %s.\n", dataTgz)

	return nil
}
