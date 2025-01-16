package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/linuxerwang/ar"
	"github.com/linuxerwang/confish"
	"github.com/linuxerwang/debmaker/debmaker"
)

type tmplVars struct {
	PkgName  string
	PostInst string
	PreRm    string
	Files    []*debmaker.FileEntry
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

func checkFlags(params *debmaker.Params) {
	if params.Version == "" {
		fmt.Println("FATAL: flag version is required.")
		os.Exit(1)
	}
	if params.Arch == "" {
		out, err := exec.Command("dpkg", "--print-architecture").Output()
		if err != nil {
			fmt.Println("Failed to run \"dpkg --print-architecture\"")
			os.Exit(1)
		}
		params.Arch = strings.TrimSpace(string(out))
	}
}

func loadSpec(params *debmaker.Params) (*debmaker.DebSpec, error) {
	var input io.Reader
	if params.Spec == "" {
		input = os.Stdin
	} else {
		h, err := os.Open(params.Spec)
		if err != nil {
			return nil, err
		}
		defer h.Close()
		input = h
	}

	b, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	vars := &tmplVars{
		PkgName:  params.PkgName,
		PostInst: params.Postinst,
		PreRm:    params.Prerm,
		Files:    []*debmaker.FileEntry{},
	}
	for _, s := range params.Files {
		it := strings.Split(s, "=")
		vars.Files = append(vars.Files, &debmaker.FileEntry{
			DebPath: strings.TrimSpace(it[0]),
			Path:    strings.TrimSpace(it[1]),
		})
	}

	for _, s := range params.Dirs {
		it := strings.Split(s, "=")
		dir := strings.TrimSpace(it[0])
		files := strings.TrimSpace(it[1])
		for _, file := range strings.Split(files, " ") {
			file = strings.TrimSpace(file)
			if len(file) == 0 {
				continue
			}
			vars.Files = append(vars.Files, &debmaker.FileEntry{
				DebPath: filepath.Join(dir, filepath.Base(file)),
				Path:    file,
			})
		}
	}

	var buf bytes.Buffer
	t := template.Must(template.New("spec").Parse(string(b)))
	if err := t.Execute(&buf, vars); err != nil {
		return nil, err
	}

	if params.Verbose {
		fmt.Println(buf.String())
	}

	debSpec := debmaker.DebSpec{}
	err = confish.Parse(strings.NewReader(buf.String()), &debSpec)
	if err != nil {
		return nil, err
	}

	debSpec.DebCtrl.Arch = params.Arch
	debSpec.DebCtrl.Version = params.Version
	debSpec.DebCtrl.Desc = params.Desc

	return &debSpec, nil
}

func addDebBinary(params *debmaker.Params, arw *ar.Writer) error {
	header := &ar.Header{
		Size:    4,
		Name:    "debian-binary",
		ModTime: time.Now(),
		Mode:    0777,
	}
	if err := arw.WriteHeader(header); err != nil {
		if params.Verbose {
			fmt.Println("Failed to write header for debian-binary.")
		}
		return err
	}
	if _, err := arw.Write([]byte("2.0\n")); err != nil {
		if params.Verbose {
			fmt.Println("Failed to add debian-binary to deb file.")
		}
		return err
	}

	if params.Verbose {
		fmt.Println("Added debian-binary to deb file.")
	}

	return nil
}

func main() {
	params := debmaker.Params{}
	flag.StringVar(&params.OutputDir, "output-dir", ".", "The output directory for the deb file, defaults to current working directory.")
	flag.StringVar(&params.Spec, "spec-file", "", "The spec file in confish format. If not specified, read from stdin.")
	flag.StringVar(&params.Desc, "desc", "", "The description of the deb file.")
	flag.StringVar(&params.Version, "version", "", "The version of the deb file.")
	flag.StringVar(&params.Arch, "arch", "", "The architecture of the deb file.")
	flag.BoolVar(&params.Verbose, "v", false, "Output verbose message.")
	flag.StringVar(&params.PkgName, "pkg-name", "", "The deb package name.")
	flag.StringVar(&params.Postinst, "postinst", "", "The postinst file.")
	flag.StringVar(&params.Prerm, "prerm", "", "The prerm file.")
	flag.Var(&params.Files, "file", "The file to be added in deb file, repeatable.")
	flag.Var(&params.Dirs, "dir", "The dir to be added in deb file, repeatable.")

	params.TmpDir = filepath.Join(os.TempDir(), "debmaker")
	if err := os.MkdirAll(params.TmpDir, os.ModePerm); err != nil {
		fmt.Printf("Failed to create tempory directory %s, %v.\n", params.TmpDir, err)
		os.Exit(1)
	}

	flag.Usage = usage
	flag.Parse()
	checkFlags(&params)

	if params.Verbose {
		fmt.Println("Arguments: ", os.Args)
		for _, s := range os.Args {
			fmt.Println("Arg: ", s)
		}
	}

	debSpec, err := loadSpec(&params)
	if err != nil {
		fmt.Printf("Failed to load spec file, %v.\n", err)
		os.Exit(1)
	}

	dfn := fmt.Sprintf("%s_%s_%s.deb", debSpec.DebCtrl.PkgName, params.Version, params.Arch)
	deb, err := os.Create(filepath.Join(params.OutputDir, dfn))
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
	if err = addDebBinary(&params, arw); err != nil {
		fmt.Printf("Failed to debian-binary to deb file, %v.\n", err)
		os.Exit(1)
	}

	// Fill file info
	if c, err := debmaker.FillFileInfo(debSpec.Content, params.Verbose); err != nil {
		fmt.Printf("Failed to collect the content tree, %v.\n", err)
		os.Exit(1)
	} else {
		debSpec.Content = c
	}

	// Add control.tar.gz
	if err = debmaker.AddControlFiles(arw, debSpec, params.TmpDir, params.Verbose); err != nil {
		fmt.Printf("Failed to add control.tar.gz to deb file, %v.\n", err)
		os.Exit(1)
	}

	// Add data.tar.gz
	if err = debmaker.AddContentFiles(arw, debSpec, params.TmpDir, params.Verbose); err != nil {
		fmt.Printf("Failed to add data.tar.gz to deb file, %v.\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created %s.\n", dfn)
}
