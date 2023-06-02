package depguard

import (
	"fmt"
	"go/ast"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// NewAnalyzer creates a new analyzer from the settings passed in
func NewAnalyzer(settings *LinterSettings) (*analysis.Analyzer, error) {
	s, err := settings.compile()
	if err != nil {
		return nil, err
	}
	analyzer := newAnalyzer(s)
	return analyzer, nil
}

func newAnalyzer(settings linterSettings) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:             "depguard",
		Doc:              "Go linter that checks if package imports are in a list of acceptable packages",
		Run:              settings.run,
		RunDespiteErrors: false,
	}
}

func (s linterSettings) run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// For Windows need to replace separator with '/'
		fileName := filepath.ToSlash(pass.Fset.Position(file.Pos()).Filename)
		lists := s.whichLists(fileName)
		for _, imp := range file.Imports {
			for _, l := range lists {
				if allowed, sugg := l.importAllowed(rawBasicLit(imp.Path)); !allowed {
					diag := analysis.Diagnostic{
						Pos:     imp.Pos(),
						End:     imp.End(),
						Message: fmt.Sprintf("import '%s' is not allowed from list '%s'", rawBasicLit(imp.Path), l.name),
					}
					if sugg != "" {
						diag.Message = fmt.Sprintf("%s: %s", diag.Message, sugg)
						diag.SuggestedFixes = append(diag.SuggestedFixes, analysis.SuggestedFix{Message: sugg})
					}
					pass.Report(diag)
				}
			}
		}
	}
	return nil, nil
}

func rawBasicLit(lit *ast.BasicLit) string {
	return strings.Trim(lit.Value, "\"")
}
