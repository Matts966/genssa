package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"reflect"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func main() {
	if len(os.Args) != 2 {
		panic("Please pass me filename 💀")
	}
	fileName := os.Args[1]
	const packageName = "fuga"

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fileName, nil, parser.AllErrors)
	if err != nil {
		log.Fatalf("Error on parser.ParseFile: %v", err)
	}
	files := []*ast.File{f}

	pkg := types.NewPackage(packageName, "")
	ssapkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset, pkg, files,
		ssa.GlobalDebug,
	)

	if err != nil {
		log.Fatal(err)
	}

	for _, v := range ssapkg.Members {
		if f := ssapkg.Func(v.Name()); f != nil {
			WriteTo(f, os.Stdout)
		}
	}
}

func WriteTo(f *ssa.Function, w io.Writer) (int64, error) {
	var buf bytes.Buffer
	WriteFunction(&buf, f)
	n, err := w.Write(buf.Bytes())
	return int64(n), err
}

// WriteFunction writes to buf a human-readable "disassembly" of f.
func WriteFunction(buf *bytes.Buffer, f *ssa.Function) {
	fmt.Fprintf(buf, "# Name: %s\n", f.String())
	if f.Pkg != nil {
		fmt.Fprintf(buf, "# Package: %s\n", f.Pkg.Pkg.Path())
	}
	if syn := f.Synthetic; syn != "" {
		fmt.Fprintln(buf, "# Synthetic:", syn)
	}
	if pos := f.Pos(); pos.IsValid() {
		fmt.Fprintf(buf, "# Location: %s\n", f.Prog.Fset.Position(pos))
	}

	if f.Parent() != nil {
		fmt.Fprintf(buf, "# Parent: %s\n", f.Parent().Name())
	}

	if f.Recover != nil {
		fmt.Fprintf(buf, "# Recover: %s\n", f.Recover)
	}

	from := f.Pkg.Pkg

	if f.FreeVars != nil {
		buf.WriteString("# Free variables:\n")
		for i, fv := range f.FreeVars {
			fmt.Fprintf(buf, "# % 3d:\t%s %s\n", i, fv.Name(), relType(fv.Type(), from))
		}
	}

	if len(f.Locals) > 0 {
		buf.WriteString("# Locals:\n")
		for i, l := range f.Locals {
			fmt.Fprintf(buf, "# % 3d:\t%s %s\n", i, l.Name(), relType(deref(l.Type()), from))
		}
	}
	writeSignature(buf, from, f.Name(), f.Signature, f.Params)
	buf.WriteString(":\n")

	if f.Blocks == nil {
		buf.WriteString("\t(external)\n")
	}

	// NB. column calculations are confused by non-ASCII
	// characters and assume 8-space tabs.
	const punchcard = 120
	const tabwidth = 8
	for _, b := range f.Blocks {
		if b == nil {
			// Corrupt CFG.
			fmt.Fprintf(buf, ".nil:\n")
			continue
		}
		n, _ := fmt.Fprintf(buf, "%d:", b.Index)
		bmsg := fmt.Sprintf("%s P:%d S:%d", b.Comment, len(b.Preds), len(b.Succs))
		fmt.Fprintf(buf, "%*s%s\n", punchcard-1-n-len(bmsg), "", bmsg)

		if false { // CFG debugging
			fmt.Fprintf(buf, "\t# CFG: %s --> %s --> %s\n", b.Preds, b, b.Succs)
		}
		for _, instr := range b.Instrs {
			buf.WriteString("\t")
			switch v := instr.(type) {
			case ssa.Value:
				l := punchcard - tabwidth
				// Left-align the instruction.
				if name := v.Name(); name != "" {
					n, _ := fmt.Fprintf(buf, "%s = ", name)
					l -= n
				}
				n, _ := buf.WriteString(instr.String())
				l -= n
				// Right-align the type if there's space.
				if t := v.Type(); t != nil {
					buf.WriteByte(' ')
					ts := "type: " + relType(t, from) + ", instruction type: " + reflect.TypeOf(instr).Elem().Name()
					l -= len(ts) + len("  ") // (spaces before and after type)
					if l > 0 {
						fmt.Fprintf(buf, "%*s", l, "")
					}
					buf.WriteString(ts)
				}
			case nil:
				// Be robust against bad transforms.
				buf.WriteString("<deleted>")
			default:
				l := punchcard - tabwidth
				// Left-align the instruction.
				n, _ := fmt.Fprintf(buf, "%s", instr.String())
				l -= n
				// Right-align the type if there's space.
				buf.WriteByte(' ')
				ts := "instruction type: " + reflect.TypeOf(instr).Elem().Name()
				l -= len(ts) + len("  ") // (spaces before and after type)
				if l > 0 {
					fmt.Fprintf(buf, "%*s", l, "")
				}
				buf.WriteString(ts)
			}
			buf.WriteString("\n")
		}
	}
	fmt.Fprintf(buf, "\n")
}

func relType(t types.Type, from *types.Package) string {
	return types.TypeString(t, types.RelativeTo(from))
}

// deref returns a pointer's element type; otherwise it returns typ.
func deref(typ types.Type) types.Type {
	if p, ok := typ.Underlying().(*types.Pointer); ok {
		return p.Elem()
	}
	return typ
}

// writeSignature writes to buf the signature sig in declaration syntax.
func writeSignature(buf *bytes.Buffer, from *types.Package, name string, sig *types.Signature, params []*ssa.Parameter) {
	buf.WriteString("func ")
	if recv := sig.Recv(); recv != nil {
		buf.WriteString("(")
		if n := params[0].Name(); n != "" {
			buf.WriteString(n)
			buf.WriteString(" ")
		}
		types.WriteType(buf, params[0].Type(), types.RelativeTo(from))
		buf.WriteString(") ")
	}
	buf.WriteString(name)
	types.WriteSignature(buf, sig, types.RelativeTo(from))
}
