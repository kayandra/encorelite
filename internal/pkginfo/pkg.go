package pkginfo

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	. "github.com/dave/jennifer/jen"
	"golang.org/x/mod/modfile"
)

type Pkg struct {
	Mod   *modfile.File
	Route []Route
}

type Route struct {
	Option ApiDirective
	Path   string
	Pkg    string
	Func   string
}

func ParsePkg(root string) (Pkg, error) {
	set := token.NewFileSet()
	mod, err := FindModFile(root)
	if err != nil {
		return Pkg{}, err
	}

	pkginfo := Pkg{Mod: mod}
	err = filepath.WalkDir(root, func(dir string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}

		pkgs, err := parser.ParseDir(set, dir, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		for _, pkg := range pkgs {
			ast.Inspect(pkg, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}

				if fn.Doc == nil {
					return true
				}

				if !fn.Name.IsExported() {
					return true
				}

				directive, err := getDirective(fn.Doc.List)
				if err != nil {
					return true
				}

				switch directive := directive.(type) {
				case ApiDirective:
					pkginfo.Route = append(pkginfo.Route, Route{
						Option: directive,
						Path:   filepath.Join(mod.Module.Mod.String(), pkg.Name),
						Pkg:    pkg.Name,
						Func:   fn.Name.String(),
					})
				}

				return true
			})

		}

		return nil
	})

	if err != nil {
		return pkginfo, err
	}

	return pkginfo, nil
}

func (r Route) Gen(s *Statement) *Statement {
	endp := r.Option.Path
	if endp == "" {
		endp = fmt.Sprintf("/%s.%s", r.Pkg, r.Func)
	}

	return s.Dot(ucfirst(r.Option.Method)).Call(
		Lit(endp),
		Qual(r.Path, r.Func),
	)
}

func FindModFile(root string) (*modfile.File, error) {
	mod := filepath.Join(root, "go.mod")
	if _, err := os.Stat(mod); errors.Is(err, fs.ErrNotExist) {
		if root == "." {
			return nil, errors.New("cannot find project root")
		}
		return FindModFile(filepath.Dir(root))
	}

	content, err := os.ReadFile(mod)
	if err != nil {
		return nil, err
	}

	return modfile.Parse(mod, content, nil)
}

func getDirective(list []*ast.Comment) (Directive, error) {
	t := list[len(list)-1].Text
	if t[1] == '/' && strings.HasPrefix(t[2:], tag) {
		return ParseDirective(t[2:])
	}
	return new(Directive), errors.New("invlid comment")
}

func ucfirst(str string) string {
	for _, v := range str {
		u := string(unicode.ToUpper(v))
		return u + strings.ToLower(str[len(u):])
	}
	return ""
}
