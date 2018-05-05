// Govanity creates HTML files containing meta tags for custom import domains.
// These files can then be served e.g. on Github Pages.
// It loads a configuration file (default ``govanity.cfg''), which defines the import
// paths and their corresponding VCS repositories.
//
// Usage:
//
//	govanity [-c cfg] [-o outdir] [-v]
//
// The config has the following layout:
//
//	[default]
//		root = <root domain>
//		repo = <url to repository>
//		vcs = <vcs>                    # default: git
//		refresh = <url redirection>    # default: https://godoc.org/*
//
//	[import "path"]
//		root = ...
//		repo = ...
//		vcs = ...
//		refresh = ...
//	[import "another/path"]
//
// If the entries for an import section are not defined, they are taken from
// the default section.  The ``repo'' and ``refresh'' entries can contain the special
// characters ``*'' and ``$''.  ``*'' is replaced by the full import path (including the
// root domain), while ``$'' is replaced by the last part of the import path.
//
// The ``refresh'' entry specifies an URL, which the generated HTML files will redirect to.
// By default, they will redirect to the corresponding godoc.org documentation.
// No redirect will be created if ``refresh'' is empty or not defined.
//
// Example config:
//
//	[default]
//		root = rtrn.io
//		repo = https://github/com/rtrn/$
//
//	[import "cmd/govanity"]
//	[import "cmd/wenv"]
//
// which will create files containing:
//
//	<meta name="go-import" content="rtrn.io/cmd/govanity git https://github.com/rtrn/govanity">
//	<meta http-equiv="refresh" content="0; url=https://godoc.org/rtrn.io/cmd/govanity">
//
// and:
//
//	<meta name="go-import" content="rtrn.io/cmd/wenv git https://github.com/rtrn/wenv">
//	<meta http-equiv="refresh" content="0; url=https://godoc.org/rtrn.io/cmd/wenv">
//
package main // import "rtrn.io/cmd/govanity"

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"gopkg.in/gcfg.v1"
)

var (
	cfgfile = flag.String("c", "govanity.cfg", "configuration file")
	outdir  = flag.String("o", ".", "output directory")
	verbose = flag.Bool("v", false, "print names of files as they are created")
)

type entry struct {
	Root    *string
	Repo    *string
	VCS     *string
	Refresh *string
}

var cfg struct {
	Default entry
	Import  map[string]*entry
}

func main() {
	log.SetPrefix("govanity: ")
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 0 {
		usage()
	}

	err := gcfg.ReadFileInto(&cfg, *cfgfile)
	ck(err)
	if cfg.Default.VCS == nil {
		s := "git"
		cfg.Default.VCS = &s
	}
	if cfg.Default.Refresh == nil {
		s := "https://godoc.org/*"
		cfg.Default.Refresh = &s
	}

	govanity()
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: govanity [flags]")
	flag.PrintDefaults()
	os.Exit(2)
}

func govanity() {
	for k, e := range cfg.Import {
		if e.Root == nil {
			e.Root = cfg.Default.Root
		}
		if e.Repo == nil {
			e.Repo = cfg.Default.Repo
		}
		if e.VCS == nil {
			e.VCS = cfg.Default.VCS
		}
		if e.Refresh == nil {
			e.Refresh = cfg.Default.Refresh
		}

		if e.Repo == nil || *e.Repo == "" {
			log.Fatalf("\"%s\": repo is empty\n", k)
		}

		imprt := k
		if e.Root != nil {
			imprt = *e.Root + "/" + imprt
		}
		split := strings.Split(k, "/")
		last := split[len(split)-1]
		r := strings.NewReplacer("*", imprt, "$", last)
		s := r.Replace(*e.Repo)
		e.Repo = &s
		if e.Refresh != nil {
			s := r.Replace(*e.Refresh)
			e.Refresh = &s
		}
	}

	for k, e := range cfg.Import {
		imprt := k
		if e.Root != nil {
			imprt = *e.Root + "/" + imprt
		}
		writeFile(imprt, *e)
	}
}

var tmpl = template.Must(template.New("main").Parse(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.Import}} {{.VCS}} {{.Repo}}">
<meta http-equiv="refresh" content="0; url={{.Refresh}}">
</head>
<body>
Redirecting to <a href="{{.Refresh}}">{{.Refresh}}</a>...
</body>
</html>
`))

var tmplnr = template.Must(template.New("main").Parse(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.Import}} {{.VCS}} {{.Repo}}">
</head>
</html>
`))

func writeFile(imprt string, e entry) {
	t := tmpl
	if e.Refresh == nil || *e.Refresh == "" {
		s := ""
		e.Refresh = &s
		t = tmplnr
	}
	d := struct {
		Import  string
		Repo    string
		VCS     string
		Refresh string
	}{imprt, *e.Repo, *e.VCS, *e.Refresh}

	var sb strings.Builder
	err := t.Execute(&sb, d)
	ck(err)
	new := sb.String()

	split := strings.SplitN(imprt, "/", 2)
	f := *outdir + "/" + split[len(split)-1]
	err = os.MkdirAll(f, os.ModePerm)
	ck(err)

	exists := false
	f += "/index.html"
	old, err := ioutil.ReadFile(f)
	if err == nil {
		exists = true
		if new == string(old) {
			return
		}
	}

	if *verbose {
		if exists {
			fmt.Printf("updating %s\n", f)
		} else {
			fmt.Printf("creating %s\n", f)
		}
	}
	err = ioutil.WriteFile(f, []byte(new), os.ModePerm)
	ck(err)
}

func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
