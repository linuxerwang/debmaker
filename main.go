package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/linuxerwang/ar"
	"github.com/linuxerwang/confish"
)

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

// DebSpec specifies the whole deb structure: control files and data files.
type DebSpec struct {
	DebCtrl *DebControl  `cfg-attr:"control"`
	Debian  []*FileEntry `cfg-attr:"debian"`
	Content []*FileEntry `cfg-attr:"content"`
}

var (
	outputDir = flag.String("output-dir", ".", "The output directory for the deb file, defaults to current working directory.")
	spec      = flag.String("spec-file", "", "The spec file in confish format. If not specified, read from stdin.")
	version   = flag.String("version", "", "The version of the deb file.")
	arch      = flag.String("arch", "", "The architecture of the deb file.")
	verbose   = flag.Bool("v", false, "Output verbose message.")

	tmpDir string
)

func init() {
	tmpDir = filepath.Join(os.TempDir(), "debmaker")
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		fmt.Println("Failed to create tempory directory %s, %v", tmpDir, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("The deb file maker.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  debmaker [-output-dir output_dir] [-spec-file spec_file] -version version [-arch architecture]")
	fmt.Println()
	flag.PrintDefaults()
	os.Exit(0)
}

func checkFlags() {
	if *version == "" {
		fmt.Println("FATAL: flag version is required.")
		os.Exit(1)
	}
	if *arch == "" {
		fmt.Println("FATAL: flag arch is required.")
		os.Exit(1)
	}
}

func loadSpec() (*DebSpec, error) {
	var input io.Reader
	if *spec == "" {
		input = os.Stdin
	} else {
		h, err := os.Open(*spec)
		if err != nil {
			return nil, err
		}
		defer h.Close()
		input = h
	}

	debSpec := DebSpec{}
	err := confish.Parse(input, &debSpec)
	if err != nil {
		return nil, err
	}

	debSpec.DebCtrl.Arch = *arch
	debSpec.DebCtrl.Version = *version

	return &debSpec, nil
}

func addDebBinary(arw *ar.Writer) error {
	header := &ar.Header{
		Size:    4,
		Name:    "debian-binary",
		ModTime: time.Now(),
		Mode:    0777,
	}
	if err := arw.WriteHeader(header); err != nil {
		if *verbose {
			fmt.Println("Failed to write header for debian-binary.")
		}
		return err
	}
	if _, err := arw.Write([]byte("2.0\n")); err != nil {
		if *verbose {
			fmt.Println("Failed to add debian-binary to deb file.")
		}
		return err
	}

	if *verbose {
		fmt.Println("Added debian-binary to deb file.")
	}

	return nil
}

func main() {
	flag.Usage = usage
	flag.Parse()
	checkFlags()

	debSpec, err := loadSpec()
	if err != nil {
		fmt.Printf("Failed to load spec file, %v.\n", err)
		os.Exit(1)
	}

	dfn := fmt.Sprintf("%s_%s_%s.deb", debSpec.DebCtrl.PkgName, *version, *arch)
	deb, err := os.Create(filepath.Join(*outputDir, dfn))
	if err != nil {
		fmt.Printf("Failed to create the deb file, %v.\n", err)
		os.Exit(1)
	}
	defer deb.Close()

	// Create ar
	arw := ar.NewWriter(deb)
	err = arw.WriteGlobalHeader()
	if err != nil {
		fmt.Printf("Failed to write global header, %v.\n", err)
		os.Exit(1)
	}

	// Add debian-binary file
	if err = addDebBinary(arw); err != nil {
		fmt.Printf("Failed to debian-binary to deb file, %v.\n", err)
		os.Exit(1)
	}

	// Fill file info
	if c, err := fillFileInfo(debSpec.Content); err != nil {
		fmt.Printf("Failed to collect the content tree, %v.\n", err)
		os.Exit(1)
	} else {
		debSpec.Content = c
	}

	// Add control.tar.gz
	if err = addControlFiles(arw, debSpec); err != nil {
		fmt.Printf("Failed to add control.tar.gz to deb file, %v.\n", err)
		os.Exit(1)
	}

	// Add data.tar.gz
	if err = addContentFiles(arw, debSpec.Content); err != nil {
		fmt.Printf("Failed to add data.tar.gz to deb file, %v.\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created %s.\n", dfn)
}
