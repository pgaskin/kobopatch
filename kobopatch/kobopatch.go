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

	yaml "gopkg.in/yaml.v2"
)

type config struct {
	Version string            `yaml:"version" json:"version"`
	In      string            `yaml:"in" json:"in"`
	Out     string            `yaml:"out" json:"out"`
	Patches map[string]string `yaml:"patches" json:"patches"`
}

func main() {
	cfgbuf, err := ioutil.ReadFile("./kobopatch.yaml")
	checkErr(err, "Could not read kobopatch.yaml")

	cfg := &config{}
	err = yaml.Unmarshal(cfgbuf, &cfg)
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
	}

	fmt.Println("This is not complete, so the output file has not been written.")

	// TODO: write logfile, delay exit on windows, write output, actually do patching, custom patch format
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
