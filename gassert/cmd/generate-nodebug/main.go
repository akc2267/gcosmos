// Command generate-nodebug processes a *_debug.go file
// to automatically and consistently generate a *_nodebug.go file,
// with functions that have matching signatures.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "USAGE: generate-nodebug xxx_debug.go\n")
		os.Exit(1)
	}

	prefix, ok := strings.CutSuffix(os.Args[1], "_debug.go")
	if !ok {
		fmt.Fprintf(os.Stderr, "Input file must end in _debug.go\n")
		os.Exit(1)
	}

	dst := prefix + "_nodebug.go"
	f, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create destination file %q: %v\n", dst, err)
		os.Exit(1)
	}
	defer f.Close()

	src, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read source file %q: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	if err := RewriteSource(os.Args[1], src, f); err != nil {
		fmt.Fprintf(os.Stderr, "rewrite failed: %v\n", err)
		os.Exit(1)
	}
}

func RewriteSource(srcName string, in []byte, w io.Writer) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, srcName, in, 0)
	if err != nil {
		return fmt.Errorf("parsing %s failed: %w", srcName, err)
	}

	goBuildDebug := []byte("//go:build debug")
	hasDebugBuildTag := false
	for _, ln := range bytes.Split(in, []byte("\n")) {
		if len(ln) < len(goBuildDebug) {
			continue
		}

		if bytes.Equal(goBuildDebug, bytes.TrimSpace(ln)) {
			hasDebugBuildTag = true
			break
		}
	}

	if !hasDebugBuildTag {
		return fmt.Errorf(
			"refusing to generate when input does not have a line exactly matching %q",
			goBuildDebug,
		)
	}

	fmt.Fprintf(w, `//go:build !debug

// Code generated by github.com/rollchains/gordian/gassert/cmd/generate-nodebug %s; DO NOT EDIT.

package %s`, srcName, f.Name.Name)

	var funcDecls []*ast.FuncDecl
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			stripFunction(fd)
			funcDecls = append(funcDecls, fd)
		}
	}

	// We've walked the file and we have the imports and function declarations.
	// Walk over the function declarations once to discover the imports.
	keepImports := scanImports(funcDecls)

	// Print the imports, if we have any.
	var startedPrintingImports, printedAnyStdlib, printedAnyThirdParty bool
	for _, imp := range f.Imports {
		var name string
		if imp.Name == nil {
			var err error
			name, err = strconv.Unquote(imp.Path.Value)
			if err != nil {
				return fmt.Errorf("failed to unquote import path %q: %w", imp.Path.Value, err)
			}

			// If there are slashes in the import path,
			// we only want what is after the final slash.
			if idx := strings.LastIndex(name, "/"); idx >= 0 {
				name = name[idx+1:]
				// TODO: maybe need to deal with hyphens in remaining name too?
			}
		} else {
			name = imp.Name.Name
		}

		if keepImports[name] {
			if !startedPrintingImports {
				if _, err := io.WriteString(w, "\n\nimport (\n"); err != nil {
					return err
				}
				startedPrintingImports = true
			}

			if printedAnyStdlib && !printedAnyThirdParty && strings.Contains(imp.Path.Value, ".") {
				// Newline to separate stdlib and third party.
				if _, err := io.WriteString(w, "\n"); err != nil {
					return err
				}
			}

			if imp.Name == nil {
				// Print the import path, which should already be quoted.
				if _, err := fmt.Fprintf(w, "\t%s\n", imp.Path.Value); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(w, "\t%s %s\n", imp.Name.Name, imp.Path.Value); err != nil {
					return err
				}
			}

			if !strings.Contains(imp.Path.Value, ".") {
				printedAnyStdlib = true
			}
		}
	}
	if startedPrintingImports {
		if _, err := io.WriteString(w, ")"); err != nil {
			return err
		}
	}

	// Now print out each (pre-stripped) function, in the same order it occurred.
	for _, fd := range funcDecls {
		if _, err := io.WriteString(w, "\n\n"); err != nil {
			return err
		}
		if err := printer.Fprint(w, fset, fd); err != nil {
			return err
		}
		// We fully removed the function body.
		// Whether we leave the body empty or put a naked return,
		// depends on whether the function has any return values.
		if fd.Type.Results == nil {
			if _, err := io.WriteString(w, " {}"); err != nil {
				return err
			}
		} else {
			// There are results, but we can already be sure they are named,
			// so a naked return suffices here.
			if _, err := io.WriteString(w, " {\n\treturn\n}"); err != nil {
				return err
			}
		}
	}

	// And finally, write the files's trailing newline.
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}

	return nil
}

func stripFunction(fd *ast.FuncDecl) {
	// We're always going to remove the function body.
	fd.Body = nil

	if fd.Recv != nil {
		for i, field := range fd.Recv.List {
			// Overwrite the field to have the names and type and nothing else
			// (no comments or tags).
			fd.Recv.List[i] = &ast.Field{
				Names: field.Names,
				Type:  field.Type,
			}
		}
	}

	// We don't modify the params at all.
	// If they are named, we keep the names for readability,
	// and if not, we don't care.

	if fd.Type.Results != nil {
		for _, field := range fd.Type.Results.List {
			// Zero-length names means the return value is unnamed.
			// That is the only time we want to replace the name with an underscore,
			// in order to use a naked return.
			// Otherwise we keep the existing names.
			if len(field.Names) == 0 {
				field.Names = []*ast.Ident{
					// We aren't marking the position, but it seems to work fine.
					&ast.Ident{Name: "_"},
				}
			}
		}
	}

	// TODO: there are probably comments that would currently leak through,
	// if they are next to fd.Type.TypeParams or fd.Type.Results.
}

func scanImports(fds []*ast.FuncDecl) map[string]bool {
	m := make(map[string]bool)
	for _, fd := range fds {
		ast.Inspect(fd.Type, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				// There is probably a better way to stringify the selector expression.
				m[fmt.Sprintf("%v", x.X)] = true
			}
			return true
		})
	}
	return m
}
