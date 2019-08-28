package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"

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
			fmt.Println(f.WriteTo(os.Stdout))
		}
		fmt.Println()
	}
}
