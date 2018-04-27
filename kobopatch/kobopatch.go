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

	"github.com/geek1011/kobopatch/patchlib"
	yaml "gopkg.in/yaml.v2"
)

var version = "unknown"

type config struct {
	Version           string            `yaml:"version" json:"version"`
	In                string            `yaml:"in" json:"in"`
	Out               string            `yaml:"out" json:"out"`
	UseNewPatchFormat bool              `yaml:"useNewPatchFormat" json:"useNewPatchFormat"`
	Patches           map[string]string `yaml:"patches" json:"patches"`
}

func main() {
	// TODO: args for verbose mode and custom config path, add logfile, add unit tests, add patch group checking, finish converting patches to new format
	fmt.Printf("kobopatch %s\n\n", version)

	cfgbuf, err := ioutil.ReadFile("./kobopatch.yaml")
	checkErr(err, "Could not read kobopatch.yaml")

	cfg := &config{}
	err = yaml.UnmarshalStrict(cfgbuf, &cfg)
	checkErr(err, "Could not parse kobopatch.yaml")

	if cfg.Version == "" || cfg.In == "" || cfg.Out == "" {
		checkErr(errors.New("version, in, and out are required"), "Could not parse kobopatch.yaml")
	}

	if !cfg.UseNewPatchFormat {
		checkErr(errors.New("only the new patch format is supported"), "Error")
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
		var pfn string
		for n, f := range cfg.Patches {
			if h.Name == "./"+f || h.Name == f {
				needsPatching = true
				pfn = n
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

		pt := patchlib.NewPatcher(fbuf)

		pf, err := newPatchFile(pfn)
		checkErr(err, "Could not read and parse patch file "+pfn)

		err = pf.ApplyTo(pt)
		checkErr(err, "Could not apply patch file "+pfn)

		fbuf = pt.GetBytes()

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

	fmt.Println("\nNote that this tool is not yet complete, so do not install it to your kobo as there may be bugs.")
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

type patchFile map[string]patch
type patch []instruction
type instruction struct {
	Enabled           *bool   `yaml:"Enabled" json:"Enabled"`
	BaseAddress       *int32  `yaml:"BaseAddress" json:"BaseAddress"`
	FindBaseAddress   *string `yaml:"FindBaseAddress" json:"FindBaseAddress"`
	FindReplaceString *struct {
		Find    string `yaml:"Find" json:"Find"`
		Replace string `yaml:"Replace" json:"Replace"`
	} `yaml:"FindReplaceString" json:"FindReplaceString"`
	ReplaceString *struct {
		Offset  int32  `yaml:"Offset" json:"Offset"`
		Find    string `yaml:"Find" json:"Find"`
		Replace string `yaml:"Replace" json:"Replace"`
	} `yaml:"ReplaceString" json:"ReplaceString"`
	ReplaceInt *struct {
		Offset  int32 `yaml:"Offset" json:"Offset"`
		Find    uint8 `yaml:"Find" json:"Find"`
		Replace uint8 `yaml:"Replace" json:"Replace"`
	} `yaml:"ReplaceInt" json:"ReplaceInt"`
	ReplaceFloat *struct {
		Offset  int32   `yaml:"Offset" json:"Offset"`
		Find    float64 `yaml:"Find" json:"Find"`
		Replace float64 `yaml:"Replace" json:"Replace"`
	} `yaml:"ReplaceFloat" json:"ReplaceFloat"`
	ReplaceBytes *struct {
		Offset  int32  `yaml:"Offset" json:"Offset"`
		Find    []byte `yaml:"Find" json:"Find"`
		Replace []byte `yaml:"Replace" json:"Replace"`
	} `yaml:"ReplaceBytes" json:"ReplaceBytes"`
}

func newPatchFile(filename string) (*patchFile, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading patch file: %v", err)
	}

	pf := &patchFile{}
	err = yaml.UnmarshalStrict(buf, &pf)
	if err != nil {
		return nil, fmt.Errorf("error parsing patch file: %v", err)
	}

	err = pf.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid patch file: %v", err)
	}

	return pf, nil
}

func (pf *patchFile) ApplyTo(pt *patchlib.Patcher) error {
	err := pf.validate()
	if err != nil {
		err = fmt.Errorf("invalid patch file: %v", err)
		fmt.Printf("  Error: %v\n", err)
		return err
	}

	num, total := 0, len(*pf)
	for n, p := range *pf {
		var err error
		num++
		pt.ResetBaseAddress()

		enabled := false
		for _, i := range p {
			if i.Enabled != nil && *i.Enabled {
				enabled = *i.Enabled
				break
			}
		}

		if !enabled {
			fmt.Printf("  [%d/%d] Skipping disabled patch `%s`\n", num, total, n)
			continue
		}

		fmt.Printf("  [%d/%d] Applying patch `%s`\n", num, total, n)

		for _, i := range p {
			switch {
			case i.Enabled != nil:
				err = nil
			case i.BaseAddress != nil:
				err = pt.BaseAddress(*i.BaseAddress)
			case i.FindBaseAddress != nil:
				err = pt.FindBaseAddressString(*i.FindBaseAddress)
			case i.ReplaceBytes != nil:
				r := *i.ReplaceBytes
				err = pt.ReplaceBytes(r.Offset, r.Find, r.Replace)
			case i.ReplaceFloat != nil:
				r := *i.ReplaceFloat
				err = pt.ReplaceFloat(r.Offset, r.Find, r.Replace)
			case i.ReplaceInt != nil:
				r := *i.ReplaceInt
				err = pt.ReplaceInt(r.Offset, r.Find, r.Replace)
			case i.ReplaceString != nil:
				r := *i.ReplaceString
				err = pt.ReplaceString(r.Offset, r.Find, r.Replace)
			case i.FindReplaceString != nil:
				r := *i.FindReplaceString
				err = pt.FindBaseAddressString(r.Find)
				if err != nil {
					err = fmt.Errorf("FindReplaceString: %v", err)
					break
				}
				err = pt.ReplaceString(0, r.Find, r.Replace)
				if err != nil {
					err = fmt.Errorf("FindReplaceString: %v", err)
					break
				}
			default:
				err = fmt.Errorf("invalid instruction: %#v", i)
			}

			if err != nil {
				fmt.Printf("    Error: could not apply patch: %v\n", err)
				return err
			}
		}
	}
	return nil
}

func (pf *patchFile) validate() error {
	for n, p := range *pf {
		ec := 0
		for _, i := range p {
			ic := 0
			if i.Enabled != nil {
				ec++
				ic++
			}
			if i.BaseAddress != nil {
				ic++
			}
			if i.FindBaseAddress != nil {
				ic++
			}
			if i.ReplaceBytes != nil {
				ic++
			}
			if i.ReplaceFloat != nil {
				ic++
			}
			if i.ReplaceInt != nil {
				ic++
			}
			if i.ReplaceString != nil {
				ic++
			}
			if i.FindReplaceString != nil {
				ic++
			}
			if ic < 1 {
				return fmt.Errorf("internal error while validating `%s` (you should report this as a bug)", n)
			}
			if ic > 1 {
				return fmt.Errorf("more than one instruction per bullet in patch `%s` (you might be missing a -)", n)
			}
		}
		if ec < 1 {
			return fmt.Errorf("no `Enabled` option in `%s`", n)
		}
		if ec > 1 {
			return fmt.Errorf("more than one `Enabled` option in `%s`", n)
		}
	}
	return nil
}
