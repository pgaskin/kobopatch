package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/pgaskin/kobopatch/patchfile"
	"github.com/pgaskin/kobopatch/patchfile/kobopatch"
	_ "github.com/pgaskin/kobopatch/patchfile/patch32lsb"
	"github.com/pgaskin/kobopatch/patchlib"

	"github.com/spf13/pflag"
	"github.com/xi2/xz"
	"gopkg.in/yaml.v3"
)

var version = "unknown"

func main() {
	help := pflag.BoolP("help", "h", false, "show this help text")
	fw := pflag.StringP("firmware", "f", "", "firmware file to be used (can also use a testdata tarball from kobopatch-patches)")
	t := pflag.BoolP("run-tests", "t", false, "test all patches (instead of running kobopatch)")
	pflag.Parse()

	if *help || pflag.NArg() > 1 {
		fmt.Fprintf(os.Stderr, "Usage: kobopatch [OPTIONS] [CONFIG_FILE]\n")
		fmt.Fprintf(os.Stderr, "\nVersion: %s\n\nOptions:\n", version)
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nIf CONFIG_FILE is not specified, kobopatch will use ./kobopatch.yaml.\n")
		os.Exit(1)
	}

	var tmp bytes.Buffer
	var logfile io.Writer = &tmp

	k := &KoboPatch{
		Logf: func(format string, a ...interface{}) {
			fmt.Printf(format+"\n", a...)
		},
		Errorf: func(format string, a ...interface{}) {
			fmt.Fprintf(os.Stderr, format+"\n", a...)
		},
		Debugf: func(format string, a ...interface{}) {
			fmt.Fprintf(logfile, format+"\n", a...)
		},
		sums: map[string]string{},
	}

	patchfile.Log = func(format string, a ...interface{}) {
		k.Debugf("          | %s", strings.ReplaceAll(fmt.Sprintf(strings.TrimRight(format, "\n"), a...), "\n", "\n          | "))
	}

	k.Logf("kobopatch %s\nhttps://github.com/pgaskin/kobopatch\n", version)
	k.Debugf("kobopatch %s\nhttps://github.com/pgaskin/kobopatch\n", version)

	conf := "kobopatch.yaml"
	if pflag.NArg() >= 1 {
		conf = pflag.Arg(0)
	}

	if *fw != "" {
		var err error
		*fw, err = filepath.Abs(*fw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not resolve path to firmware file: %v\n", err)
			os.Exit(1)
		}
	}

	k.Logf("Loading configuration from %s", conf)
	if conf == "-" {
		err := k.LoadConfig(os.Stdin)
		if err != nil {
			k.Errorf("Error: could not load config file from stdin: %v", err)
			os.Exit(1)
			return
		}
	} else {
		os.Chdir(filepath.Dir(conf))
		f, err := os.Open(conf)
		if err != nil {
			k.Errorf("Error: could not load config file: %v", err)
			os.Exit(1)
			return
		}
		err = k.LoadConfig(f)
		if err != nil {
			k.Errorf("Error: could not load config file: %v", err)
			os.Exit(1)
			return
		}
		f.Close()
	}

	f, err := os.Create(k.Config.Log)
	if err != nil {
		k.Errorf("Error: could not create log file")
		os.Exit(1)
		return
	}
	defer f.Close()
	f.Write(tmp.Bytes())
	logfile = f

	if *fw != "" {
		k.d("firmware file overridden from command line: %s", *fw)
		k.Config.In = *fw
	}

	if *t {
		if res, err := k.RunPatchTests(); err != nil {
			k.Errorf("Error: could not apply patches: %v", err)
			os.Exit(1)
			return
		} else {
			errs := []string{}
			for pfn, ps := range res {
				for pn, err := range ps {
					if err != nil {
						errs = append(errs, fmt.Sprintf("%s: %s: %v", pfn, pn, err))
					}
				}
			}
			if len(errs) > 0 {
				k.l("\nErrors:\n  %s", strings.Join(errs, "\n  "))
				if runtime.GOOS == "windows" {
					fmt.Printf("\n\nWaiting 60 seconds because runnning on Windows\n")
					time.Sleep(time.Second * 60)
				}
				os.Exit(1)
			}
		}
		fmt.Println("\nAll patches applied successfully.")
		if runtime.GOOS == "windows" {
			fmt.Printf("\n\nWaiting 60 seconds because runnning on Windows\n")
			time.Sleep(time.Second * 60)
		}
		os.Exit(0)
	}

	k.OutputInit()

	if err := k.ApplyPatches(); err != nil {
		k.Errorf("Error: could not apply patches: %v", err)
		os.Exit(1)
		return
	}

	if err := k.ApplyTranslations(); err != nil {
		k.Errorf("Error: could not apply translations: %v", err)
		os.Exit(1)
		return
	}

	if err := k.ApplyFiles(); err != nil {
		k.Errorf("Error: could not apply additional files: %v", err)
		os.Exit(1)
		return
	}

	if err := k.WriteOutput(); err != nil {
		k.Errorf("Error: could not write output: %v", err)
		os.Exit(1)
		return
	}

	fmt.Printf("\nSuccessfully saved patched KoboRoot.tgz to %s. Remember to make sure your kobo is running the target firmware version before patching.\n", k.Config.Out)

	if runtime.GOOS == "windows" {
		fmt.Printf("\n\nWaiting 60 seconds because runnning on Windows\n")
		time.Sleep(time.Second * 60)
	}
}

type KoboPatch struct {
	Config *Config

	outBuf             bytes.Buffer
	outTar             *tar.Writer
	outGZ              *gzip.Writer
	outTarExpectedSize int64
	sums               map[string]string

	Logf   func(format string, a ...interface{}) // displayed to user
	Errorf func(format string, a ...interface{}) // displayed to user
	Debugf func(format string, a ...interface{}) // for verbose logging
}

type Config struct {
	Version      string
	In           string
	Out          string
	Log          string
	PatchFormat  string `yaml:"patchFormat"` // DEPRECATED: now detected from extension; .patch -> p32lsb, .yaml -> kobopatch
	Patches      map[string]string
	Overrides    map[string]map[string]bool
	Lrelease     string
	Translations map[string]string
	Files        map[string]stringSlice
}

func (k *KoboPatch) OutputInit() {
	k.d("\n\nKoboPatch::OutputInit")
	k.outBuf.Reset()
	k.outGZ = gzip.NewWriter(&k.outBuf)
	k.outTar = tar.NewWriter(k.outGZ)
}

func (k *KoboPatch) WriteOutput() error {
	k.d("\n\nKoboPatch::WriteOutput")

	k.d("Removing old output tgz '%s'", k.Config.Out)
	os.Remove(k.Config.Out)

	k.d("Closing tar")
	if err := k.outTar.Close(); err != nil {
		k.d("--> %v", err)
		return wrap(err, "could not finalize output tar.gz")
	}
	k.outTar = nil

	k.d("Closing gz")
	if err := k.outGZ.Close(); err != nil {
		k.d("--> %v", err)
		return wrap(err, "could not finalize output tar.gz")
	}
	k.outGZ = nil

	k.d("Writing buf to output '%s'", k.Config.Out)
	if err := ioutil.WriteFile(k.Config.Out, k.outBuf.Bytes(), 0644); err != nil {
		k.d("--> %v", err)
		return wrap(err, "could not write output tar.gz")
	}

	k.l("\nChecking patched KoboRoot.tgz for consistency")
	k.d("Checking patched KoboRoot.tgz for consistency")

	f, err := os.Open(k.Config.Out)
	if err != nil {
		k.d("--> %v", err)
		return wrap(err, "could not open output for reading")
	}

	zr, err := gzip.NewReader(f)
	if err != nil {
		k.d("--> %v", err)
		return wrap(err, "could not open output gz")
	}

	tr := tar.NewReader(zr)

	var sum int64
	for h, err := tr.Next(); err != io.EOF; h, err = tr.Next() {
		sum += h.Size
	}

	k.d("sum:%d == expected:%d", sum, k.outTarExpectedSize)
	if sum != k.outTarExpectedSize {
		k.d("--> size mismatch")
		return fmt.Errorf("size mismatch: expected %d, got %d (please report this)", k.outTarExpectedSize, sum)
	}

	k.d("\nsha1 checksums:")
	for f, s := range k.sums {
		k.d("  %s %s", s, f)
	}

	return nil
}

func (k *KoboPatch) LoadConfig(r io.Reader) error {
	k.d("\n\nKoboPatch::LoadConfig")

	k.d("reading config file from %v", reflect.TypeOf(r))
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		k.d("--> %v", err)
		return wrap(err, "error reading config")
	}

	k.d("unmarshaling yaml")
	dec := yaml.NewDecoder(bytes.NewReader(buf))
	dec.KnownFields(true)
	err = dec.Decode(&k.Config)
	if err != nil {
		k.d("--> %v", err)
		return wrap(err, "error reading kobopatch.yaml")
	}

	if k.Config.Version == "" || k.Config.In == "" || k.Config.Out == "" || k.Config.Log == "" {
		err = errors.New("invalid kobopatch.yaml: version, in, out, and log are required")
		k.d("--> %v", err)
		return err
	}

	if _, ok := patchfile.GetFormat(k.Config.PatchFormat); !ok {
		err = fmt.Errorf("invalid patch format '%s', expected one of %s", k.Config.PatchFormat, strings.Join(patchfile.GetFormats(), ", "))
		k.d("--> %v", err)
		return err
	}

	k.dp("  | ", "%s", jm(k.Config))
	return nil
}

func (k *KoboPatch) ApplyPatches() error {
	k.d("\n\nKoboPatch::ApplyPatches")

	tr, closeAll, err := k.openIn()
	if err != nil {
		return err
	}
	defer closeAll()

	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			k.d("--> could not read entry from tgz: %v", err)
			return wrap(err, "could not read input firmware")
		}

		patchfiles := []string{}
		for n, f := range k.Config.Patches {
			if h.Name == "./"+f || h.Name == f || filepath.Base(f) == h.Name {
				if filepath.Base(f) == h.Name { // from testdata tarball
					h.Name = "./" + f
				}
				patchfiles = append(patchfiles, n)
			}
		}

		if len(patchfiles) < 1 {
			continue
		}

		k.d("    patching entry name:'%s' size:%d mode:'%v' typeflag:'%v' with files: %s", h.Name, h.Size, h.Mode, h.Typeflag, strings.Join(patchfiles, ", "))
		k.l("\nPatching %s", h.Name)

		if h.Typeflag != tar.TypeReg {
			k.d("    --> could not patch: not a regular file")
			return fmt.Errorf("could not patch file '%s': not a regular file", h.Name)
		}

		k.d("        reading entry contents")
		buf, err := ioutil.ReadAll(tr)
		if err != nil {
			k.d("    --> could not patch: could not read contents: %v", err)
			return wrap(err, "could not patch file '%s': could not read contents", h.Name)
		}

		pt := patchlib.NewPatcher(buf)

		for _, pfn := range patchfiles {
			k.d("        loading patch file '%s' (detected format %s)", pfn, getFormat(pfn))
			ps, err := patchfile.ReadFromFile(getFormat(pfn), pfn)
			if err != nil {
				k.d("        --> %v", err)
				return wrap(err, "could not load patch file '%s'", pfn)
			}

			for ofn, o := range k.Config.Overrides {
				if ofn != pfn || o == nil || len(o) < 1 {
					continue
				}
				k.l("  Applying overrides")
				k.d("        applying overrides")
				for on, os := range o {
					if os {
						k.l("    ENABLE  `%s`", on)
					} else {
						k.l("    DISABLE `%s`", on)
					}
					k.d("            override %s -> enabled:%t", on, os)
					if err := ps.SetEnabled(on, os); err != nil {
						k.d("            --> %v", err)
						return wrap(err, "could not override enabled for patch '%s'", on)
					}
				}
			}

			k.d("        validating patch file")
			if err := ps.Validate(); err != nil {
				k.d("        --> %v", err)
				return wrap(err, "invalid patch file '%s'", pfn)
			}

			k.d("        applying patch file")
			if err := ps.ApplyTo(pt); err != nil {
				k.d("        --> %v", err)
				return wrap(err, "error applying patch file '%s'", pfn)
			}
		}

		fbuf := pt.GetBytes()
		k.outTarExpectedSize += h.Size
		k.d("        patched file - orig:%d new:%d", h.Size, len(fbuf))

		k.d("        copying new header to output tar - size:%d mode:'%v'", len(fbuf), h.Mode)
		// Preserve attributes (VERY IMPORTANT)
		err = k.outTar.WriteHeader(&tar.Header{
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
		if err != nil {
			k.d("        --> %v", err)
			return wrap(err, "could not write new file header to patched KoboRoot.tgz")
		}

		k.d("        writing patched file to tar writer")
		if i, err := k.outTar.Write(fbuf); err != nil {
			k.d("        --> %v", err)
			return wrap(err, "error writing new file to patched KoboRoot.tgz")
		} else if i != len(fbuf) {
			k.d("        --> error writing new file to patched KoboRoot.tgz")
			return errors.New("error writing new file to patched KoboRoot.tgz")
		}

		k.sums[h.Name] = fmt.Sprintf("%x", sha1.Sum(fbuf))
	}

	return nil
}

func (k *KoboPatch) ApplyTranslations() error {
	k.d("\n\nKoboPatch::ApplyTranslations")
	if len(k.Config.Translations) >= 1 {
		k.l("\nProcessing translations")
		k.d("looking for lrelease in config")
		lr := k.Config.Lrelease
		var err error
		if lr == "" {
			k.d("looking for lrelease in path")
			lr, err = exec.LookPath("lrelease")
			if lr == "" {
				k.d("looking for lrelease.exe in path")
				lr, err = exec.LookPath("lrelease.exe")
				if err != nil {
					k.d("--> %v", err)
					return wrap(err, "could not find lrelease (part of QT Linguist)")
				}
			}
		} else if lr, err = exec.LookPath(lr); err != nil {
			k.d("--> %v", err)
			return wrap(err, "could not find lrelease (part of QT Linguist)")
		}

		for ts, qm := range k.Config.Translations {
			k.l("  LRELEASE  %s", ts)
			k.d("    processing '%s' -> '%s'", ts, qm)
			if !strings.HasPrefix(qm, "usr/local/Kobo/translations/") {
				err = errors.New("output for translation must start with usr/local/Kobo/translations/")
				k.d("    --> %v", err)
				return wrap(err, "could not process translation")
			}

			k.d("        creating temp dir for lrelease")
			td, err := ioutil.TempDir(os.TempDir(), "lrelease-qm")
			if err != nil {
				k.d("        --> %v", err)
				return wrap(err, "could not make temp dir for lrelease")
			}

			tf := filepath.Join(td, "out.qm")

			cmd := exec.Command(lr, ts, "-qm", tf)
			var outbuf, errbuf bytes.Buffer
			cmd.Stdout, cmd.Stderr = &outbuf, &errbuf

			err = cmd.Run()
			k.dp("          | ", "lrelease stdout: %s", outbuf.String())
			k.dp("          | ", "lrelease stderr: %s", errbuf.String())
			if err != nil {
				k.e(errbuf.String())
				os.RemoveAll(td)
				k.d("        --> %v", err)
				return wrap(err, "error running lrelease")
			}

			k.d("        reading generated qm '%s'", ts)
			buf, err := ioutil.ReadFile(tf)
			if err != nil {
				k.d("        --> %v", err)
				return wrap(err, "could not read generated qm file")
			}
			os.RemoveAll(td)

			k.d("        writing header")
			err = k.outTar.WriteHeader(&tar.Header{
				Typeflag: tar.TypeReg,
				Name:     "./" + qm,
				Mode:     0777,
				Uid:      0,
				Gid:      0,
				ModTime:  time.Now(),
				Size:     int64(len(buf)),
			})
			if err != nil {
				k.d("    --> %v", err)
				return wrap(err, "could not write translation file to KoboRoot.tgz")
			}

			k.d("        writing file")
			if i, err := k.outTar.Write(buf); err != nil {
				k.d("    --> %v", err)
				return wrap(err, "error writing translation file to KoboRoot.tgz")
			} else if i != len(buf) {
				k.d("    --> error writing translation file to KoboRoot.tgz")
				return errors.New("error writing translation file to KoboRoot.tgz")
			}
			k.outTarExpectedSize += int64(len(buf))
		}
	}
	return nil
}

func (k *KoboPatch) ApplyFiles() error {
	k.d("\n\nKoboPatch::ApplyFiles")
	if len(k.Config.Files) >= 1 {
		k.l("\nAdding additional files")
		for src, dests := range k.Config.Files {
			for _, dest := range dests {
				k.l("  ADD  %-35s  TO  %s", src, dest)
				k.d("    %s -> %s", src, dest)
				if strings.HasPrefix(dest, "/") {
					k.d("    --> destination must not start with a slash")
					return errors.New("could not add file: destination must not start with a slash")
				}

				k.d("        reading file")
				buf, err := ioutil.ReadFile(src)
				if err != nil {
					k.d("    --> %v", err)
					return wrap(err, "could not read additional file '%s'", src)
				}

				k.d("        writing header")
				err = k.outTar.WriteHeader(&tar.Header{
					Typeflag: tar.TypeReg,
					Name:     "./" + dest,
					Mode:     0777,
					Uid:      0,
					Gid:      0,
					ModTime:  time.Now(),
					Size:     int64(len(buf)),
				})
				if err != nil {
					k.d("    --> %v", err)
					return wrap(err, "could not write additional file to KoboRoot.tgz")
				}

				k.d("        writing file")
				if i, err := k.outTar.Write(buf); err != nil {
					k.d("    --> %v", err)
					return wrap(err, "error writing additional file to KoboRoot.tgz")
				} else if i != len(buf) {
					k.d("    --> error writing additional file to KoboRoot.tgz")
					return errors.New("error writing additional file to KoboRoot.tgz")
				}
				k.outTarExpectedSize += int64(len(buf))
			}
		}
	}
	return nil
}

func (k *KoboPatch) RunPatchTests() (map[string]map[string]error, error) {
	k.d("\n\nKoboPatch::RunPatchTests")

	res := map[string]map[string]error{}

	tr, closeAll, err := k.openIn()
	if err != nil {
		return nil, err
	}
	defer closeAll()

	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			k.d("--> could not read entry from tgz: %v", err)
			return nil, wrap(err, "could not read input firmware")
		}

		patchfiles := []string{}
		for n, f := range k.Config.Patches {
			if h.Name == "./"+f || h.Name == f || filepath.Base(f) == h.Name {
				patchfiles = append(patchfiles, n)
			}
		}

		if len(patchfiles) < 1 {
			continue
		}

		k.d("    patching entry name:'%s' size:%d mode:'%v' typeflag:'%v' with files: %s", h.Name, h.Size, h.Mode, h.Typeflag, strings.Join(patchfiles, ", "))
		k.l("\nPatching %s", h.Name)

		if h.Typeflag != tar.TypeReg {
			k.d("    --> could not patch: not a regular file")
			return nil, fmt.Errorf("could not patch file '%s': not a regular file", h.Name)
		}

		k.d("        reading entry contents")
		buf, err := ioutil.ReadAll(tr)
		if err != nil {
			k.d("    --> could not patch: could not read contents: %v", err)
			return nil, wrap(err, "could not patch file '%s': could not read contents", h.Name)
		}
		getBuf := func() []byte {
			nbuf := make([]byte, len(buf))
			copy(nbuf, buf)
			return nbuf
		}

		for _, pfn := range patchfiles {
			k.d("        loading patch file '%s' (detected format %s)", pfn, getFormat(pfn))
			if getFormat(pfn) != "kobopatch" {
				k.d("        --> format not kobopatch")
				return nil, errors.New("patch testing only works with kobopatch format patches")
			}
			ps, err := patchfile.ReadFromFile(getFormat(pfn), pfn)
			if err != nil {
				k.d("        --> %v", err)
				return nil, wrap(err, "could not load patch file '%s'", pfn)
			}

			k.d("        validating patch file")
			if err := ps.Validate(); err != nil {
				k.d("        --> %v", err)
				return nil, wrap(err, "invalid patch file '%s'", pfn)
			}

			res[pfn] = map[string]error{}

			sortedNames := reflect.ValueOf(ps).Interface().(*kobopatch.PatchSet).SortedNames()

			errs := map[string]error{}
			for _, name := range sortedNames {
				fmt.Printf(" -  %s", name)
				var err error
				for _, pname := range sortedNames {
					if err = ps.SetEnabled(pname, pname == name); err != nil {
						break
					}
				}
				if err != nil {
					fmt.Printf("\r ✕  %s\n", name)
					errs[name] = err
					res[pfn][name] = err
					continue
				}
				out := os.Stdout
				os.Stdout = nil
				if err := ps.ApplyTo(patchlib.NewPatcher(getBuf())); err != nil {
					os.Stdout = out
					fmt.Printf("\r ✕  %s\n", name)
					errs[name] = err
					res[pfn][name] = err
					continue
				}
				os.Stdout = out
				if err := ps.SetEnabled(name, false); err != nil {
					fmt.Printf("\r ✕  %s\n", name)
					errs[name] = err
					res[pfn][name] = err
					continue
				}
				fmt.Printf("\r ✔  %s\n", name)
				res[pfn][name] = nil
			}
		}
	}

	k.dp("  | ", "%s", jm(res))

	return res, nil
}

func (k *KoboPatch) openIn() (*tar.Reader, func(), error) {
	k.d("    KoboPatch::openIn")
	closeReaders := func() {}
	var tbr io.Reader
	if strings.HasSuffix(k.Config.In, ".tar.xz") {
		k.l("Reading input firmware testdata tarball")
		k.d("        Opening testdata tarball '%s'", k.Config.In)

		f, err := os.Open(k.Config.In)
		if err != nil {
			k.d("        --> %v", err)
			return nil, closeReaders, wrap(err, "could not open firmware tarball")
		}

		xzr, err := xz.NewReader(f, 0)
		if err != nil {
			k.d("        --> %v", err)
			f.Close()
			return nil, closeReaders, wrap(err, "could not open firmware tarball as xz")
		}
		tbr = xzr
		closeReaders = func() { f.Close() }
	} else {
		k.l("Reading input firmware zip")
		k.d("        Opening firmware zip '%s'", k.Config.In)

		zr, err := zip.OpenReader(k.Config.In)
		if err != nil {
			k.d("        --> %v", err)
			return nil, closeReaders, wrap(err, "could not open firmware zip")
		}

		k.d("        Looking for KoboRoot.tgz in zip")
		var kr io.ReadCloser
		for _, f := range zr.File {
			k.d("        --> found %s", f.Name)
			if f.Name == "KoboRoot.tgz" {
				k.d("        -->    opening KoboRoot.tgz")
				kr, err = f.Open()
				if err != nil {
					k.d("        -->    --> %v", err)
					return nil, closeReaders, wrap(err, "could not open KoboRoot.tgz in firmware zip")
				}
				break
			}
		}
		if kr == nil {
			k.d("        --> could not find KoboRoot.tgz")
			return nil, closeReaders, errors.New("could not find KoboRoot.tgz")
		}

		k.d("        Opening gzip reader")
		gzr, err := gzip.NewReader(kr)
		if err != nil {
			k.d("        --> %v", err)
			kr.Close()
			return nil, closeReaders, wrap(err, "could not decompress KoboRoot.tgz")
		}
		tbr = gzr
		closeReaders = func() {
			gzr.Close()
			kr.Close()
		}
	}

	k.d("        Creating tar reader")
	return tar.NewReader(tbr), closeReaders, nil
}

func (k *KoboPatch) l(format string, a ...interface{}) {
	if k.Logf != nil {
		k.Logf(format, a...)
	}
}

func (k *KoboPatch) e(format string, a ...interface{}) {
	if k.Errorf != nil {
		k.Errorf(format, a...)
	}
}

func (k *KoboPatch) d(format string, a ...interface{}) {
	if k.Debugf != nil {
		k.Debugf(format, a...)
	}
}

func (k *KoboPatch) dp(prefix string, format string, a ...interface{}) {
	k.d("%s%s", prefix, strings.ReplaceAll(fmt.Sprintf(format, a...), "\n", "\n"+prefix))
}

func wrap(err error, format string, a ...interface{}) error {
	return fmt.Errorf("%s: %v", fmt.Sprintf(format, a...), err)
}

func jm(v interface{}) string {
	if buf, err := json.MarshalIndent(v, "", "    "); err == nil {
		return string(buf)
	}
	return ""
}

func getFormat(filename string) string {
	f := strings.TrimLeft(filepath.Ext(filename), ".")
	f = strings.ReplaceAll(f, "patch", "patch32lsb")
	f = strings.ReplaceAll(f, "yaml", "kobopatch")
	return f
}

// stringArray forces strings to become arrays during yaml decoding.
type stringSlice []string

// UnmarshalYAML unmarshals a stringArray.
func (a *stringSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strings []string
	if err := unmarshal(&strings); err != nil {
		var str string
		if err := unmarshal(&str); err != nil {
			return err
		}
		*a = []string{str}
	} else {
		*a = strings
	}
	return nil
}
