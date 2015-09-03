package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/linuxerwang/ar"
)

const dataTgz = "data.tar.gz"

func addContentFiles(arw *ar.Writer, files []*FileEntry) error {
	if err := createDataTgz(files); err != nil {
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
		if *verbose {
			fmt.Printf("Failed to write header for %s.\n", dataTgz)
		}
		return err
	}
	if _, err := io.Copy(arw, f); err != nil {
		if *verbose {
			fmt.Println("Failed to copy %s to deb file.", dataTgz)
		}
		return err
	}

	fmt.Printf("Added %s to deb file.\n", dataTgz)

	return nil
}

func createDataTgz(files []*FileEntry) error {
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

	// Write other data files
	for _, df := range files {
		info := df.FileInfo

		hdr := &tar.Header{
			Name:    df.DebPath,
			Mode:    0755,
			ModTime: info.ModTime(),
		}
		if info.IsDir() {
			hdr.Size = 0
			hdr.Typeflag |= tar.TypeDir

			if !strings.HasSuffix(hdr.Name, "/") {
				hdr.Name += "/"
			}
		} else {
			hdr.Size = info.Size()
		}

		if err := w.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			if *verbose {
				fmt.Printf("Added directory %s to %s.\n", df.DebPath, dataTgz)
			}
			continue
		}

		// Note that the opened file has to be closed explicitly
		in, err := os.Open(df.Path)
		if err != nil {
			if *verbose {
				fmt.Printf("Failed to open file %s.\n", df.Path)
			}
			return err
		}

		_, err = io.Copy(w, in)
		if err != nil {
			if *verbose {
				fmt.Printf("Failed to copy content from %s to %s.\n", df.Path, dataTgz)
			}
			in.Close()
			return err
		}

		if err = in.Close(); err != nil {
			if *verbose {
				fmt.Printf("Failed to close file %s.\n", df.Path)
			}
			return err
		}

		if *verbose {
			fmt.Printf("Added file %s to %s.\n", df.DebPath, dataTgz)
		}
	}

	fmt.Printf("Created temporary file %s.\n", dataTgz)

	return nil
}
