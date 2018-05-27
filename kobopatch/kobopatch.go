// Command kobopatch is an all-in-one tool to apply patches to a kobo update zip.
package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/pkg/errors"

	"github.com/geek1011/kobopatch/patchfile"
	_ "github.com/geek1011/kobopatch/patchfile/kobopatch"
	_ "github.com/geek1011/kobopatch/patchfile/patch32lsb"
	"github.com/geek1011/kobopatch/patchlib"
	yaml "gopkg.in/yaml.v2"
)

var version = "unknown"

type config struct {
	Version     string                     `yaml:"version" json:"version"`
	In          string                     `yaml:"in" json:"in"`
	Out         string                     `yaml:"out" json:"out"`
	Log         string                     `yaml:"log" json:"log"`
	PatchFormat string                     `yaml:"patchFormat" json:"patchFormat"`
	Patches     map[string]string          `yaml:"patches" json:"patches"`
	Overrides   map[string]map[string]bool `yaml:"overrides" json:"overrides"`
}

var log = func(format string, a ...interface{}) {}

func main() {
	fmt.Printf("kobopatch %s\n", version)
	fmt.Printf("https://github.com/geek1011/kobopatch\n\n")

	fmt.Printf("Reading config file (kobopatch.yaml)\n")

	cfgbuf, err := ioutil.ReadFile("./kobopatch.yaml")
	checkErr(err, "Could not read kobopatch.yaml")

	cfg := &config{}
	err = yaml.UnmarshalStrict(cfgbuf, &cfg)
	checkErr(err, "Could not parse kobopatch.yaml")

	if cfg.Version == "" || cfg.In == "" || cfg.Out == "" || cfg.Log == "" {
		checkErr(errors.New("version, in, out, and log are required"), "Could not parse kobopatch.yaml")
	}

	_, ok := patchfile.GetFormat(cfg.PatchFormat)
	if !ok {
		checkErr(errors.New("invalid patch format"), "Error")
	}

	logf, err := os.Create(cfg.Log)
	checkErr(err, "Could not open and truncate log file")
	defer logf.Close()

	log = func(format string, a ...interface{}) {
		fmt.Fprintf(logf, format, a...)
	}
	patchfile.Log = func(format string, a ...interface{}) {
		fmt.Fprintf(logf, "        "+format, a...)
	}

	log("config: %#v\n", cfg)

	d, _ := os.Getwd()
	log("kobopatch %s\n\ndir:%s\ncfg: %#v\n\n", version, d, cfg)

	fmt.Printf("Opening input file\n")

	log("opening zip\n")
	zipr, err := zip.OpenReader(cfg.In)
	checkErr(err, "Could not open input file")
	defer zipr.Close()

	log("searching for KoboRoot.tgz\n")
	var tgzr io.ReadCloser
	for _, f := range zipr.File {
		log("  file: %s\n", f.Name)
		if f.Name == "KoboRoot.tgz" {
			log("found KoboRoot.tgz, opening\n")
			tgzr, err = f.Open()
			checkErr(err, "Could not open KoboRoot.tgz")
			break
		}
	}
	if tgzr == nil {
		log("KoboRoot.tgz reader empty so KoboRoot.tgz not in zip\n")
		checkErr(errors.New("no such file in zip"), "Could not open KoboRoot.tgz")
	}
	defer tgzr.Close()

	log("creating new gzip reader for tgz\n")
	tdr, err := gzip.NewReader(tgzr)
	checkErr(err, "Could not decompress KoboRoot.tgz")
	defer tdr.Close()

	log("creating new tar reader for gzip reader for tgz\n")
	tr := tar.NewReader(tdr)
	checkErr(err, "Could not read KoboRoot.tgz as tar archive")

	log("creating new buffer for output\n")
	var outw bytes.Buffer
	outzw := gzip.NewWriter(&outw)
	defer outzw.Close()

	log("creating new tar writer for output buffer\n")
	outtw := tar.NewWriter(outzw)
	defer outtw.Close()

	var expectedSizeSum int64

	log("looping over files from source tgz\n")
	for {
		log("  reading entry\n")
		h, err := tr.Next()
		if err == io.EOF {
			err = nil
			break
		}
		checkErr(err, "Could not read entry from KoboRoot.tgz")
		log("    entry: %s - size:%d, mode:%v\n", h.Name, h.Size, h.Mode)

		log("    checking if entry needs patching\n")
		var needsPatching bool
		var pfn string
		for n, f := range cfg.Patches {
			if h.Name == "./"+f || h.Name == f {
				log("    entry needs patching\n")
				needsPatching = true
				pfn = n
				break
			}
		}

		if !needsPatching {
			log("    entry does not need patching\n")
			continue
		}

		log("    checking type before patching - typeflag: %v\n", h.Typeflag)
		fmt.Printf("Patching %s\n", h.Name)

		if h.Typeflag != tar.TypeReg {
			checkErr(errors.New("not a regular file"), "Could not patch file")
		}

		log("    reading entry contents\n")
		fbuf, err := ioutil.ReadAll(tr)
		checkErr(err, "Could not read file contents from KoboRoot.tgz")

		pt := patchlib.NewPatcher(fbuf)

		log("    loading patch file: %s\n", pfn)
		ps, err := patchfile.ReadFromFile(cfg.PatchFormat, pfn)
		checkErr(err, "Could not read and parse patch file "+pfn)

		for ofn, o := range cfg.Overrides {
			if ofn != pfn || o == nil || len(o) < 1 {
				continue
			}
			fmt.Printf("  Applying overrides from config\n")
			for on, os := range o {
				log("    override: %s -> %t\n", on, os)
				fmt.Printf("    '%s' -> enabled:%t\n", on, os)
				err := ps.SetEnabled(on, os)
				checkErr(err, "Could not override patch '"+on+"'")
			}
		}

		log("    validating patch file\n")
		err = ps.Validate()
		checkErr(err, "Invalid patch file "+pfn)

		log("    applying patch file\n")
		err = ps.ApplyTo(pt)
		checkErr(err, "Could not apply patch file "+pfn)

		fbuf = pt.GetBytes()

		expectedSizeSum += h.Size

		log("    copying new header to output tar - size:%d, mode:%v\n", len(fbuf), h.Mode)
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
			Format:     h.Format,
		})
		checkErr(err, "Could not write new header to patched KoboRoot.tgz")

		log("    writing patched binary to output\n")
		i, err := outtw.Write(fbuf)
		checkErr(err, "Could not write new file to patched KoboRoot.tgz")
		if i != len(fbuf) {
			checkErr(errors.New("could not write whole file"), "Could not write new file to patched KoboRoot.tgz")
		}
	}

	fmt.Printf("Writing patched KoboRoot.tgz\n")

	log("removing old output tgz: %s\n", cfg.Out)
	os.Remove(cfg.Out)

	log("flushing output tar writer to buffer\n")
	err = outtw.Close()
	checkErr(err, "Could not finish writing patched tar")
	time.Sleep(time.Millisecond * 500)

	log("flushing output gzip writer to buffer\n")
	err = outzw.Close()
	checkErr(err, "Could not finish writing compressed patched tar")
	time.Sleep(time.Millisecond * 500)

	log("writing buffer to output file\n")
	err = ioutil.WriteFile(cfg.Out, outw.Bytes(), 0644)
	checkErr(err, "Could not write patched KoboRoot.tgz")

	fmt.Printf("Checking patched KoboRoot.tgz for consistency\n")
	log("checking consistency\n")

	log("opening out as read-only\n")
	checkr, err := os.Open(cfg.Out)
	checkErr(err, "Could not open patched tgz")
	defer checkr.Close()

	log("creating gzip reader\n")
	checkzr, err := gzip.NewReader(checkr)
	checkErr(err, "Could not open patched tgz")
	defer checkzr.Close()

	log("creating tar reader\n")
	checktr := tar.NewReader(checkzr)
	checkErr(err, "Could not open patched tgz")

	var sizeSum int64
	for {
		h, err := checktr.Next()
		if err == io.EOF {
			break
		}
		log("  reading entry: %s: %d\n", h.Name, h.Size)
		for _, f := range cfg.Patches {
			if h.Name == "./"+f || h.Name == f {
				sizeSum += h.Size
				log("  matched, added %d to sum, sum:%s\n", h.Size, sizeSum)
				break
			}
		}
	}
	if expectedSizeSum != sizeSum {
		checkErr(errors.Errorf("size mismatch: expected %d, got %d. (please report this as a bug)", expectedSizeSum, sizeSum), "Error checking patched KoboRoot.tgz for consistency")
	}

	log("patch success\n")
	fmt.Printf("Successfully saved patched KoboRoot.tgz to %s. Remember to make sure your kobo is running the target firmware version before patching.\n", cfg.Out)

	if runtime.GOOS == "windows" {
		fmt.Printf("\n\nWaiting 60 seconds because runnning on Windows\n")
		time.Sleep(time.Second * 60)
	}
}

func checkErr(err error, msg string) {
	if err == nil {
		return
	}
	if msg != "" {
		log("Fatal: %s: %v\n", msg, err)
		fmt.Fprintf(os.Stderr, "Fatal: %s: %v\n", msg, err)
	} else {
		log("Fatal: %v\n", err)
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
	}
	if runtime.GOOS == "windows" {
		fmt.Printf("\n\nWaiting 60 seconds because runnning on Windows\n")
		time.Sleep(time.Second * 60)
	}
	os.Exit(1)
}
