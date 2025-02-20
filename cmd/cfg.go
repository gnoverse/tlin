package cmd

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/gnolang/tlin/internal/analysis/cfg"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// variable for flags
var (
	funcName string
	output   string
)

var cfgCmd = &cobra.Command{
	Use:   "cfg [paths...]",
	Short: "Run control flow graph analysis",
	Long: `Outputs the Control Flow Graph (CFG) of the specified function or generates a GraphViz file.
Example) tlin cfg --func MyFunction *.go`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("error: Please provide file or directory paths")
			os.Exit(1)
		}
		// timeout is a global variable declared in root.go
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		runCFGAnalysis(ctx, logger, args, funcName, output)
	},
}

func init() {
	cfgCmd.Flags().StringVar(&funcName, "func", "", "Function name for CFG analysis")
	cfgCmd.Flags().StringVarP(&output, "output", "o", "", "Output path for rendered GraphViz file")
}

func runCFGAnalysis(_ context.Context, logger *zap.Logger, paths []string, funcName string, output string) {
	functionFound := false
	for _, path := range paths {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			logger.Error("Failed to parse file", zap.String("path", path), zap.Error(err))
			continue
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if fn.Name.Name == funcName {
					cfgGraph := cfg.FromFunc(fn)
					var buf strings.Builder
					cfgGraph.PrintDot(&buf, fset, func(n ast.Stmt) string { return "" })
					if output != "" {
						err := cfg.RenderToGraphVizFile([]byte(buf.String()), output)
						if err != nil {
							logger.Error("Failed to render CFG to GraphViz file", zap.Error(err))
						} else {
							fmt.Printf("GraphViz file created: %s\n", output)
						}
					} else {
						fmt.Printf("CFG for function %s in file %s:\n%s\n", funcName, path, buf.String())
					}
					functionFound = true
					return
				}
			}
		}
	}

	if !functionFound {
		fmt.Printf("Function not found: %s\n", funcName)
	}
}
