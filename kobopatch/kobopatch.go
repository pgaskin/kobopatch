package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type config struct {
	Version string            `yaml:"version" json:"version"`
	In      string            `yaml:"in" json:"in"`
	Out     string            `yaml:"out" json:"out"`
	Patches map[string]string `yaml:"patches" json:"patches"`
}

func main() {
	// TODO: args for verbose mode, custom config path

	cfgbuf, err := ioutil.ReadFile("./kobopatch.yaml")
	checkErr(err, "Could not read kobopatch.yaml")

	cfg := &config{}
	err = yaml.UnmarshalStrict(cfgbuf, &cfg)
	checkErr(err, "Could not parse kobopatch.yaml")

	if cfg.Version == "" || cfg.In == "" || cfg.Out == "" {
		checkErr(errors.New("version, in, and out are required"), "Could not parse kobopatch.yaml")
	}

	zipr, err := zip.OpenReader(cfg.In)
	checkErr(err, "Could not open input file")
	defer zipr.Close()

	var tgzr io.ReadCloser
	for _, f := range zipr.File {
		if f.Name == "KoboRoot.tgz" {
			tgzr, err = f.Open()
			checkErr(err, "Could not open KoboRoot.tgz")
			break
		}
	}
	if tgzr == nil {
		checkErr(errors.New("no such file in zip"), "Could not open KoboRoot.tgz")
	}
	defer tgzr.Close()

	tdr, err := gzip.NewReader(tgzr)
	checkErr(err, "Could not decompress KoboRoot.tgz")
	defer tdr.Close()

	tr := tar.NewReader(tdr)
	checkErr(err, "Could not read KoboRoot.tgz as tar archive")

	var outw bytes.Buffer
	outzw := gzip.NewWriter(&outw)
	defer outzw.Close()

	outtw := tar.NewWriter(outzw)
	defer outtw.Close()

	for {
		h, err := tr.Next()
		if err == io.EOF {
			err = nil
			break
		}
		checkErr(err, "Could not read entry from KoboRoot.tgz")

		var needsPatching bool
		for _, f := range cfg.Patches {
			if h.Name == "./"+f || h.Name == f {
				needsPatching = true
				break
			}
		}

		if !needsPatching {
			continue
		}

		fmt.Printf("Patching %s\n", h.Name)

		if h.Typeflag != tar.TypeReg {
			checkErr(errors.New("not a regular file"), "Could not patch file")
		}

		fbuf, err := ioutil.ReadAll(tr)
		checkErr(err, "Could not read file contents from KoboRoot.tgz")

		// TODO: patching stuff here
		// TODO: custom patch format

		// Preserve attributes (VERY IMPORTANT)
		err = outtw.WriteHeader(&tar.Header{
			Typeflag:   h.Typeflag,
			Name:       h.Name,
			Mode:       h.Mode,
			Uid:        h.Uid,
			Gid:        h.Gid,
			ModTime:    time.Now(),
			Uname:      h.Uname,
			Gname:      h.Gname,
			PAXRecords: h.PAXRecords,
			Size:       int64(len(fbuf)),
		})
		checkErr(err, "Could not write new header to patched KoboRoot.tgz")

		i, err := outtw.Write(fbuf)
		checkErr(err, "Could not write new file to patched KoboRoot.tgz")
		if i != len(fbuf) {
			checkErr(errors.New("could not write whole file"), "Could not write new file to patched KoboRoot.tgz")
		}
	}

	os.Remove(cfg.Out)

	err = outtw.Flush()
	checkErr(err, "Could not finish writing patched tar")
	err = outtw.Close()
	checkErr(err, "Could not finish writing patched tar")

	err = outzw.Flush()
	checkErr(err, "Could not finish writing compressed patched tar")
	err = outzw.Close()
	checkErr(err, "Could not finish writing compressed patched tar")

	err = ioutil.WriteFile(cfg.Out, outw.Bytes(), 0644)
	checkErr(err, "Could not write patched KoboRoot.tgz")

	fmt.Printf("Successfully saved patched KoboRoot.tgz to %s\n", cfg.Out)

	fmt.Println("\nNote that this tool is not complete yet, so the files were not actually patched.")
}

func fataln(n int, msg string) {
	fmt.Fprintf(os.Stderr, "    Fatal: Line %d: %s\n", n, msg)
	os.Exit(1)
}

func checkErr(err error, msg string) {
	if err == nil {
		return
	}
	if msg != "" {
		fmt.Fprintf(os.Stderr, "Fatal: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
	}
	os.Exit(1)
}
