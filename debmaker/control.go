package debmaker

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/linuxerwang/ar"
)

const ctrlTgz = "control.tar.gz"

func AddControlFiles(arw *ar.Writer, debSpec *DebSpec, tmpDir string, verbose bool) error {
	if err := createControlTgz(debSpec, tmpDir, verbose); err != nil {
		return err
	}

	tmpConl := filepath.Join(tmpDir, ctrlTgz)

	info, err := os.Stat(tmpConl)
	if err != nil {
		return err
	}

	f, err := os.Open(tmpConl)
	if err != nil {
		return err
	}
	defer f.Close()

	hdr := &ar.Header{
		Size:    info.Size(),
		Name:    ctrlTgz,
		ModTime: time.Now(),
		Mode:    0777,
	}
	if err := arw.WriteHeader(hdr); err != nil {
		if verbose {
			fmt.Printf("Failed to write header for %s\n.", ctrlTgz)
		}
		return err
	}
	if _, err := io.Copy(arw, f); err != nil {
		if verbose {
			fmt.Printf("Failed to copy %s to deb file: %v.\n", ctrlTgz, err)
		}
		return err
	}

	fmt.Printf("Added %s to deb file.\n", ctrlTgz)

	return nil
}

func createControlTgz(debSpec *DebSpec, tmpDir string, verbose bool) error {
	tmpConl := filepath.Join(tmpDir, ctrlTgz)
	f, err := os.Create(tmpConl)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	w := tar.NewWriter(gz)
	defer w.Close()

	md5sums, size := createMd5sums(debSpec.Content)

	// Write control
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Package: %s\n", debSpec.DebCtrl.PkgName))
	buf.WriteString(fmt.Sprintf("Version: %s\n", debSpec.DebCtrl.Version))
	buf.WriteString(fmt.Sprintf("Architecture: %s\n", debSpec.DebCtrl.Arch))
	buf.WriteString(fmt.Sprintf("Maintainer: %s\n", debSpec.DebCtrl.Maintainer))
	buf.WriteString(fmt.Sprintf("Installed-Size: %d\n", size))
	buf.WriteString(fmt.Sprintf("Description: %s\n", debSpec.DebCtrl.Desc))
	for k, v := range debSpec.DebCtrl.Attrs {
		buf.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}

	hdr := &tar.Header{
		Name: "control",
		Mode: 0755,
		Size: int64(buf.Len()),
	}
	if err := w.WriteHeader(hdr); err != nil {
		if verbose {
			fmt.Printf("Failed to write header for %s\n.", ctrlTgz)
		}
		return err
	}
	_, err = io.WriteString(w, buf.String())
	if err != nil {
		if verbose {
			fmt.Printf("Failed to write header for %s\n.", ctrlTgz)
		}
		return err
	}
	if verbose {
		fmt.Printf("Added file control to %s.\n", ctrlTgz)
	}

	// Write md5sums
	hdr = &tar.Header{
		Name: "md5sums",
		Mode: 0755,
		Size: int64(len(md5sums)),
	}
	if err := w.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.WriteString(w, md5sums)
	if err != nil {
		if verbose {
			fmt.Println("Failed to create md5sums.")
		}
		return err
	}
	if verbose {
		fmt.Printf("Added file md5sums to %s.\n", ctrlTgz)
	}

	// Write debian files
	for _, cf := range debSpec.Debian {
		info, err := os.Stat(cf.Path)
		if err != nil {
			return err
		}

		hdr = &tar.Header{
			Name:    cf.DebPath,
			Mode:    int64(info.Mode()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		if err := w.WriteHeader(hdr); err != nil {
			if verbose {
				fmt.Printf("Failed to write header for %s\n.", ctrlTgz)
			}
			return err
		}

		// Note that the opened file has to be closed explicitly
		in, err := os.Open(cf.Path)
		if err != nil {
			if verbose {
				fmt.Printf("Failed to open file %s\n.", cf.Path)
			}
			return err
		}

		_, err = io.Copy(w, in)
		if err != nil {
			if verbose {
				fmt.Printf("Failed to copy content from file %s to %s.\n", cf.Path, ctrlTgz)
			}
			in.Close()
			return err
		}

		if err = in.Close(); err != nil {
			return err
		}

		if verbose {
			fmt.Printf("Added file %s to %s.\n", cf.DebPath, ctrlTgz)
		}
	}

	fmt.Printf("Created temporary file %s.\n", ctrlTgz)

	return nil
}

func createMd5sums(files []*FileEntry) (string, int64) {
	var buf bytes.Buffer
	var size int64
	for _, f := range files {
		if f.Md5sum != "" {
			size += f.FileInfo.Size()
			buf.WriteString(fmt.Sprintf("%s  %s\n", f.Md5sum, f.DebPath))
		}
	}

	return buf.String(), size
}
