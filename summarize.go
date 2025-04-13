package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CodeSummary holds parsed information for a Go file.
type CodeSummary struct {
	Filename           string
	Package            string
	Types              []TypeDecl
	Functions          []FuncDecl
	Imports            []string
	Lines              int
	CommentLines       int
	LongFunctions      []FuncDecl
	AvgComplexity      float64
	GodocCoverage      float64
	MaxFunctionDepth   int
	MaintainabilityIdx float64
}

// TypeDecl represents a type declaration.
type TypeDecl struct {
	Name       string
	Comment    string
	Definition string
	Exported   bool
}

// FuncDecl represents a function or method declaration.
type FuncDecl struct {
	Name       string
	Comment    string
	Signature  string
	LineCount  int
	Complexity int
	MaxDepth   int
	Exported   bool
}

// ProjectOverview holds aggregated project metrics.
type ProjectOverview struct {
	TotalFiles      int
	TotalLines      int
	TotalFunctions  int
	TotalLongFuncs  int
	AvgCommentRatio float64
	AvgComplexity   float64
	GodocCoverage   float64
	PackageCount    int
	DependencyCount int
	ProjectHealth   float64
	RiskyFiles      int
	EffortHours     float64
	PackageMetrics  map[string]PackageMetric
}

// PackageMetric holds metrics for a package.
type PackageMetric struct {
	FileCount     int
	LineCount     int
	ImportCount   int
	CouplingCount int
}

// scanDirectory recursively finds all .go files (excluding test files).
func scanDirectory(root string) ([]string, error) {
	var goFiles []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go") {
			goFiles = append(goFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning directory: %w", err)
	}
	return goFiles, nil
}

// parseFile parses a Go file and extracts detailed metrics.
func parseFile(filename string) (CodeSummary, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return CodeSummary{}, fmt.Errorf("parsing file %s: %w", filename, err)
	}

	summary := CodeSummary{Filename: filename, Package: f.Name.Name}

	// Count lines and comments
	if err := countLines(&summary, filename); err != nil {
		return CodeSummary{}, err
	}

	// Collect imports
	summary.Imports = collectImports(f.Imports)

	// Extract types and functions
	metrics, err := extractDeclarations(f, fset)
	if err != nil {
		return CodeSummary{}, err
	}

	summary.Types = metrics.types
	summary.Functions = metrics.functions
	summary.LongFunctions = metrics.longFunctions
	summary.AvgComplexity = metrics.avgComplexity
	summary.GodocCoverage = metrics.godocCoverage
	summary.MaxFunctionDepth = metrics.maxFunctionDepth
	summary.MaintainabilityIdx = calculateMaintainability(summary.Lines, summary.CommentLines, summary.AvgComplexity)

	return summary, nil
}

// countLines counts total and comment lines in a file.
func countLines(summary *CodeSummary, filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", filename, err)
	}
	lines := strings.Split(string(content), "\n")
	summary.Lines = len(lines)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			summary.CommentLines++
		}
	}
	return nil
}

// collectImports extracts import paths from AST.
func collectImports(imports []*ast.ImportSpec) []string {
	var result []string
	for _, imp := range imports {
		if imp.Path != nil {
			impPath := strings.Trim(imp.Path.Value, `"`)
			result = append(result, impPath)
		}
	}
	return result
}

// declMetrics holds metrics extracted from declarations.
type declMetrics struct {
	types            []TypeDecl
	functions        []FuncDecl
	longFunctions    []FuncDecl
	avgComplexity    float64
	godocCoverage    float64
	maxFunctionDepth int
}

// extractDeclarations processes type and function declarations.
func extractDeclarations(f *ast.File, fset *token.FileSet) (declMetrics, error) {
	var metrics declMetrics
	var comments []*ast.CommentGroup
	var exportedTypes, documentedTypes, exportedFuncs, documentedFuncs, totalComplexity int

	// Collect comments
	for _, cg := range f.Comments {
		comments = append(comments, cg)
	}

	getComment := func(pos token.Pos) string {
		for _, cg := range comments {
			if cg.End() < pos && cg.End() >= pos-2 {
				return strings.TrimSpace(cg.Text())
			}
		}
		return ""
	}

	// Extract types
	for _, decl := range f.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				typeSpec := spec.(*ast.TypeSpec)
				def, err := formatTypeDef(typeSpec)
				if err != nil {
					continue
				}
				isExported := ast.IsExported(typeSpec.Name.Name)
				if isExported {
					exportedTypes++
					if getComment(typeSpec.Pos()) != "" {
						documentedTypes++
					}
				}
				metrics.types = append(metrics.types, TypeDecl{
					Name:       typeSpec.Name.Name,
					Comment:    getComment(typeSpec.Pos()),
					Definition: def,
					Exported:   isExported,
				})
			}
		}
	}

	// Extract functions
	for _, decl := range f.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			lineCount := fset.Position(funcDecl.End()).Line - fset.Position(funcDecl.Pos()).Line + 1
			complexity, maxDepth := calcFuncMetrics(funcDecl)
			totalComplexity += complexity
			isExported := ast.IsExported(funcDecl.Name.Name)
			if isExported {
				exportedFuncs++
				if getComment(funcDecl.Pos()) != "" {
					documentedFuncs++
				}
			}
			if maxDepth > metrics.maxFunctionDepth {
				metrics.maxFunctionDepth = maxDepth
			}

			sig := formatFuncSignature(funcDecl)
			funcDeclData := FuncDecl{
				Name:       funcDecl.Name.Name,
				Comment:    getComment(funcDecl.Pos()),
				Signature:  sig,
				LineCount:  lineCount,
				Complexity: complexity,
				MaxDepth:   maxDepth,
				Exported:   isExported,
			}
			metrics.functions = append(metrics.functions, funcDeclData)
			if lineCount > 50 {
				metrics.longFunctions = append(metrics.longFunctions, funcDeclData)
			}
		}
	}

	// Calculate metrics
	if len(metrics.functions) > 0 {
		metrics.avgComplexity = float64(totalComplexity) / float64(len(metrics.functions))
	}
	totalExported := exportedTypes + exportedFuncs
	if totalExported > 0 {
		metrics.godocCoverage = float64(documentedTypes+documentedFuncs) / float64(totalExported) * 100
	}

	return metrics, nil
}

// formatTypeDef formats a type definition.
func formatTypeDef(typeSpec *ast.TypeSpec) (string, error) {
	var def strings.Builder
	switch t := typeSpec.Type.(type) {
	case *ast.StructType:
		def.WriteString("struct {\n")
		for _, field := range t.Fields.List {
			for _, name := range field.Names {
				def.WriteString(fmt.Sprintf("\t%s %s\n", name.Name, field.Type))
			}
		}
		def.WriteString("}")
	case *ast.InterfaceType:
		def.WriteString("interface {\n")
		for _, method := range t.Methods.List {
			for _, name := range method.Names {
				def.WriteString(fmt.Sprintf("\t%s %s\n", name.Name, method.Type))
			}
		}
		def.WriteString("}")
	default:
		return "", fmt.Errorf("unsupported type: %T", t)
	}
	return fmt.Sprintf("type %s %s", typeSpec.Name.Name, def.String()), nil
}

// calcFuncMetrics calculates cyclomatic complexity and max depth.
func calcFuncMetrics(funcDecl *ast.FuncDecl) (complexity, maxDepth int) {
	if funcDecl.Body == nil {
		return 1, 0
	}
	complexity = 1
	currentDepth := 0
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.SelectStmt:
			complexity++
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		case *ast.BlockStmt:
			if n != funcDecl.Body {
				currentDepth++
				if currentDepth > maxDepth {
					maxDepth = currentDepth
				}
			}
		}
		return true
	})
	return complexity, maxDepth
}

// formatFuncSignature formats a function signature.
func formatFuncSignature(funcDecl *ast.FuncDecl) string {
	var sig strings.Builder
	sig.WriteString("func ")
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recv := funcDecl.Recv.List[0]
		if len(recv.Names) > 0 {
			sig.WriteString(fmt.Sprintf("(%s %s) ", recv.Names[0].Name, recv.Type))
		} else {
			sig.WriteString(fmt.Sprintf("(%s) ", recv.Type))
		}
	}
	sig.WriteString(funcDecl.Name.Name)
	sig.WriteString("(")
	for i, param := range funcDecl.Type.Params.List {
		if i > 0 {
			sig.WriteString(", ")
		}
		for j, name := range param.Names {
			if j > 0 {
				sig.WriteString(", ")
			}
			sig.WriteString(fmt.Sprintf("%s %s", name.Name, param.Type))
		}
	}
	sig.WriteString(")")
	if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
		sig.WriteString(" ")
		if len(funcDecl.Type.Results.List) > 1 {
			sig.WriteString("(")
		}
		for i, result := range funcDecl.Type.Results.List {
			if i > 0 {
				sig.WriteString(", ")
			}
			sig.WriteString(fmt.Sprintf("%s", result.Type))
		}
		if len(funcDecl.Type.Results.List) > 1 {
			sig.WriteString(")")
		}
	}
	return sig.String()
}

// calculateMaintainability computes the maintainability index.
func calculateMaintainability(lines, commentLines int, avgComplexity float64) float64 {
	if lines == 0 {
		return 100.0
	}
	commentRatio := float64(commentLines) / float64(lines)
	idx := 100 - (float64(lines)/100 + avgComplexity*2 - commentRatio*50)
	if idx < 0 {
		return 0
	}
	if idx > 100 {
		return 100
	}
	return idx
}

// generateMarkdown writes the Markdown summary.
func generateMarkdown(summaries []CodeSummary, outputPath string) error {
	var b strings.Builder
	overview := computeProjectOverview(summaries)

	b.WriteString("📝 # Go Code Summary\n\n")
	b.WriteString("📊 ## Project Overview\n\n")
	if overview.TotalFiles == 0 {
		b.WriteString("No Go files found.\n\n")
	} else {
		b.WriteString(fmt.Sprintf("- 📂 Files Processed: %d\n", overview.TotalFiles))
		b.WriteString(fmt.Sprintf("- 📏 Total Lines of Code: %d\n", overview.TotalLines))
		b.WriteString(fmt.Sprintf("- 🛠️ Total Functions: %d\n", overview.TotalFunctions))
		b.WriteString(fmt.Sprintf("- ⚠️ Long Functions (>50 lines): %d\n", overview.TotalLongFuncs))
		b.WriteString(fmt.Sprintf("- 📜 Average Comment-to-Code Ratio: %.2f%%\n", overview.AvgCommentRatio))
		b.WriteString(fmt.Sprintf("- 🧠 Average Function Complexity: %.2f\n", overview.AvgComplexity))
		b.WriteString(fmt.Sprintf("- 📖 Godoc Coverage: %.2f%%\n", overview.GodocCoverage))
		b.WriteString(fmt.Sprintf("- 📦 Packages: %d\n", overview.PackageCount))
		b.WriteString(fmt.Sprintf("- 🔗 External Dependencies: %d\n", overview.DependencyCount))
		b.WriteString(fmt.Sprintf("- 🏥 Project Health Score: %.2f/100\n", overview.ProjectHealth))
		b.WriteString(fmt.Sprintf("- 🚨 Risky Files: %d\n", overview.RiskyFiles))
		b.WriteString(fmt.Sprintf("- ⏰ Estimated Refactoring Effort: %.2f hours\n\n", overview.EffortHours))

		b.WriteString("📦 ### Package Breakdown\n\n")
		if len(overview.PackageMetrics) == 0 {
			b.WriteString("No packages found.\n\n")
		} else {
			b.WriteString("| Package | Files | Lines | Imports | Coupling |\n")
			b.WriteString("|---------|-------|-------|---------|----------|\n")
			for pkg, metric := range overview.PackageMetrics {
				b.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d |\n", pkg, metric.FileCount, metric.LineCount, metric.ImportCount, metric.CouplingCount))
			}
			b.WriteString("\n")
		}
	}

	for _, summary := range summaries {
		commentRatio := 0.0
		if summary.Lines > 0 {
			commentRatio = float64(summary.CommentLines) / float64(summary.Lines) * 100
		}
		maxFuncLines := 0
		for _, f := range summary.Functions {
			if f.LineCount > maxFuncLines {
				maxFuncLines = f.LineCount
			}
		}
		b.WriteString(fmt.Sprintf("📂 ## %s (%s)\n\n", summary.Filename, summary.Package))
		b.WriteString("📈 **Metrics**:\n")
		b.WriteString(fmt.Sprintf("- 📏 Lines of Code: %d\n", summary.Lines))
		b.WriteString(fmt.Sprintf("- 🛠️ Number of Functions: %d\n", len(summary.Functions)))
		b.WriteString(fmt.Sprintf("- 📏 Largest Function: %d lines\n", maxFuncLines))
		b.WriteString(fmt.Sprintf("- ⚠️ Long Functions (>50 lines): %d\n", len(summary.LongFunctions)))
		b.WriteString(fmt.Sprintf("- 📜 Comment-to-Code Ratio: %.2f%%\n", commentRatio))
		b.WriteString(fmt.Sprintf("- 🧠 Average Function Complexity: %.2f\n", summary.AvgComplexity))
		b.WriteString(fmt.Sprintf("- 📖 Godoc Coverage: %.2f%%\n", summary.GodocCoverage))
		b.WriteString(fmt.Sprintf("- 🔲 Max Function Depth: %d\n", summary.MaxFunctionDepth))
		b.WriteString(fmt.Sprintf("- 🛡️ Maintainability Index: %.2f\n", summary.MaintainabilityIdx))
		b.WriteString(fmt.Sprintf("- 🔗 External Dependencies: %d\n\n", len(summary.Imports)))

		if len(summary.Types) > 0 {
			b.WriteString("🏗️ ### Types\n\n")
			for _, t := range summary.Types {
				if t.Comment != "" {
					b.WriteString(fmt.Sprintf("%s\n\n", t.Comment))
				}
				b.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", t.Definition))
			}
		}

		if len(summary.Functions) > 0 {
			b.WriteString("🛠️ ### Functions\n\n")
			for _, f := range summary.Functions {
				if f.Comment != "" {
					b.WriteString(fmt.Sprintf("%s\n\n", f.Comment))
				}
				b.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", f.Signature))
			}
		}
	}

	return os.WriteFile(outputPath, []byte(b.String()), 0644)
}

// generateHTML writes the HTML summary with visualizations.
func generateHTML(summaries []CodeSummary, outputPath string) error {
	const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go Code Summary</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        pre { background-color: #1f2937; color: #e5e7eb; padding: 1rem; border-radius: 0.5rem; }
        code { font-family: monospace; }
    </style>
</head>
<body class="bg-gray-100 font-sans">
    <div class="container mx-auto p-4">
        <h1 class="text-3xl font-bold mb-4">📝 Go Code Summary</h1>
        <h2 class="text-2xl font-semibold mb-2">📊 Project Overview</h2>
        {{if eq .ProjectOverview.TotalFiles 0}}
        <p>No Go files found.</p>
        {{else}}
        <ul class="list-disc ml-6 mb-4">
            <li>📂 Files Processed: {{.ProjectOverview.TotalFiles}}</li>
            <li>📏 Total Lines of Code: {{.ProjectOverview.TotalLines}}</li>
            <li>🛠️ Total Functions: {{.ProjectOverview.TotalFunctions}}</li>
            <li>⚠️ Long Functions (>50 lines): {{.ProjectOverview.TotalLongFuncs}}</li>
            <li>📜 Average Comment-to-Code Ratio: {{printf "%.2f" .ProjectOverview.AvgCommentRatio}}%</li>
            <li>🧠 Average Function Complexity: {{printf "%.2f" .ProjectOverview.AvgComplexity}}</li>
            <li>📖 Godoc Coverage: {{printf "%.2f" .ProjectOverview.GodocCoverage}}%</li>
            <li>📦 Packages: {{.ProjectOverview.PackageCount}}</li>
            <li>🔗 External Dependencies: {{.ProjectOverview.DependencyCount}}</li>
            <li>🏥 Project Health Score: {{printf "%.2f" .ProjectOverview.ProjectHealth}}/100</li>
            <li>🚨 Risky Files: {{.ProjectOverview.RiskyFiles}}</li>
            <li>⏰ Estimated Refactoring Effort: {{printf "%.2f" .ProjectOverview.EffortHours}} hours</li>
        </ul>
        <h3 class="text-lg font-medium mb-2">📦 Package Breakdown</h3>
        {{if .ProjectOverview.PackageMetrics}}
        <canvas id="packageChart" class="mb-4"></canvas>
        <script>
            const ctx = document.getElementById('packageChart').getContext('2d');
            new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: [{{range $pkg, $metric := .ProjectOverview.PackageMetrics}}'{{$pkg}}',{{end}}],
                    datasets: [{
                        label: 'File Count',
                        data: [{{range $pkg, $metric := .ProjectOverview.PackageMetrics}}{{$metric.FileCount}},{{end}}],
                        backgroundColor: '#3b82f6',
                    }, {
                        label: 'Line Count',
                        data: [{{range $pkg, $metric := .ProjectOverview.PackageMetrics}}{{$metric.LineCount}},{{end}}],
                        backgroundColor: '#10b981',
                    }]
                },
                options: { scales: { y: { beginAtZero: true } } }
            });
        </script>
        {{else}}
        <p>No packages found.</p>
        {{end}}
        {{end}}
        {{range .Summaries}}
        <details class="mb-4 bg-white rounded-lg shadow">
            <summary class="p-4 text-xl font-semibold cursor-pointer">📂 {{.Filename}} ({{.Package}})</summary>
            <div class="p-4">
                <h3 class="text-lg font-medium">📈 Metrics</h3>
                <ul class="list-disc ml-6 mb-4">
                    <li>📏 Lines of Code: {{.Lines}}</li>
                    <li>🛠️ Number of Functions: {{len .Functions}}</li>
                    <li>📏 Largest Function: {{.MaxFuncLines}} lines</li>
                    <li>⚠️ Long Functions (>50 lines): {{len .LongFunctions}}</li>
                    <li>📜 Comment-to-Code Ratio: {{printf "%.2f" .CommentRatio}}%</li>
                    <li>🧠 Average Function Complexity: {{printf "%.2f" .AvgComplexity}}</li>
                    <li>📖 Godoc Coverage: {{printf "%.2f" .GodocCoverage}}%</li>
                    <li>🔲 Max Function Depth: {{.MaxFunctionDepth}}</li>
                    <li>🛡️ Maintainability Index: {{printf "%.2f" .MaintainabilityIdx}}</li>
                    <li>🔗 External Dependencies: {{len .Imports}}</li>
                </ul>
                {{if .Types}}
                <h3 class="text-lg font-medium">🏗️ Types</h3>
                {{range .Types}}
                {{if .Comment}}
                <p class="mb-2">{{.Comment}}</p>
                {{end}}
                <pre><code>{{.Definition}}</code></pre>
                {{end}}
                {{end}}
                {{if .Functions}}
                <h3 class="text-lg font-medium mt-4">🛠️ Functions</h3>
                {{range .Functions}}
                {{if .Comment}}
                <p class="mb-2">{{.Comment}}</p>
                {{end}}
                <pre><code>{{.Signature}}</code></pre>
                {{end}}
                {{end}}
            </div>
        </details>
        {{end}}
    </div>
</body>
</html>`

	type TemplateData struct {
		Summaries []struct {
			CodeSummary
			MaxFuncLines int
			CommentRatio float64
		}
		ProjectOverview
	}

	overview := computeProjectOverview(summaries)
	data := TemplateData{ProjectOverview: overview}
	for _, s := range summaries {
		commentRatio := 0.0
		if s.Lines > 0 {
			commentRatio = float64(s.CommentLines) / float64(s.Lines) * 100
		}
		maxFuncLines := 0
		for _, f := range s.Functions {
			if f.LineCount > maxFuncLines {
				maxFuncLines = f.LineCount
			}
		}
		data.Summaries = append(data.Summaries, struct {
			CodeSummary
			MaxFuncLines int
			CommentRatio float64
		}{
			CodeSummary:  s,
			MaxFuncLines: maxFuncLines,
			CommentRatio: commentRatio,
		})
	}

	tmpl, err := template.New("summary").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parsing HTML template: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating HTML file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing HTML template: %w", err)
	}
	return nil
}

// computeProjectOverview aggregates project-wide metrics.
func computeProjectOverview(summaries []CodeSummary) ProjectOverview {
	overview := ProjectOverview{PackageMetrics: make(map[string]PackageMetric)}
	var totalCommentRatio, totalComplexity, totalGodoc float64
	uniqueDeps := make(map[string]bool)
	packageCoupling := make(map[string]map[string]bool)

	for _, s := range summaries {
		overview.TotalFiles++
		overview.TotalLines += s.Lines
		overview.TotalFunctions += len(s.Functions)
		overview.TotalLongFuncs += len(s.LongFunctions)
		if s.Lines > 0 {
			totalCommentRatio += float64(s.CommentLines) / float64(s.Lines) * 100
		}
		totalComplexity += s.AvgComplexity
		totalGodoc += s.GodocCoverage

		// Package metrics
		pkgMetric := overview.PackageMetrics[s.Package]
		pkgMetric.FileCount++
		pkgMetric.LineCount += s.Lines
		pkgMetric.ImportCount += len(s.Imports)
		overview.PackageMetrics[s.Package] = pkgMetric

		// Dependencies and coupling
		for _, imp := range s.Imports {
			uniqueDeps[imp] = true
			for _, otherSum := range summaries {
				if otherSum.Package == imp {
					if _, exists := packageCoupling[s.Package]; !exists {
						packageCoupling[s.Package] = make(map[string]bool)
					}
					packageCoupling[s.Package][imp] = true
				}
			}
		}

		// Risky files
		if s.AvgComplexity > 5 || s.GodocCoverage < 50 || len(s.LongFunctions) > 3 {
			overview.RiskyFiles++
		}
	}

	overview.PackageCount = len(overview.PackageMetrics)
	overview.DependencyCount = len(uniqueDeps)
	if overview.TotalFiles > 0 {
		overview.AvgCommentRatio = totalCommentRatio / float64(overview.TotalFiles)
		overview.AvgComplexity = totalComplexity / float64(overview.TotalFiles)
		overview.GodocCoverage = totalGodoc / float64(overview.TotalFiles)
	}

	// Project Health Score
	if overview.TotalFiles > 0 {
		health := overview.AvgCommentRatio/100*30 +
			overview.GodocCoverage/100*30 +
			(1-float64(overview.TotalLongFuncs)/float64(overview.TotalFunctions+1))*20 +
			(10-overview.AvgComplexity)/10*20
		overview.ProjectHealth = health
		if overview.ProjectHealth > 100 {
			overview.ProjectHealth = 100
		}
		if overview.ProjectHealth < 0 {
			overview.ProjectHealth = 0
		}
	}

	// Effort estimate
	overview.EffortHours = float64(overview.TotalLines)/100*0.5 +
		overview.AvgComplexity*float64(overview.TotalFunctions)*0.2 +
		float64(overview.TotalLongFuncs)*5

	// Package coupling
	for pkg, metric := range overview.PackageMetrics {
		metric.CouplingCount = len(packageCoupling[pkg])
		overview.PackageMetrics[pkg] = metric
	}

	return overview
}

// generateJSON writes the JSON summary.
func generateJSON(summaries []CodeSummary, outputPath string) error {
	type JSONSummary struct {
		Filename           string     `json:"filename"`
		Package            string     `json:"package"`
		Types              []TypeDecl `json:"types"`
		Functions          []FuncDecl `json:"functions"`
		Imports            []string   `json:"imports"`
		Lines              int        `json:"lines"`
		CommentLines       int        `json:"comment_lines"`
		MaxFuncLines       int        `json:"largest_function_lines"`
		CommentRatio       float64    `json:"comment_ratio"`
		LongFunctions      []FuncDecl `json:"long_functions"`
		AvgComplexity      float64    `json:"avg_complexity"`
		GodocCoverage      float64    `json:"godoc_coverage"`
		MaxFunctionDepth   int        `json:"max_function_depth"`
		MaintainabilityIdx float64    `json:"maintainability_index"`
	}

	overview := computeProjectOverview(summaries)
	type JSONOutput struct {
		Overview ProjectOverview `json:"overview"`
		Files    []JSONSummary   `json:"files"`
	}

	var jsonData JSONOutput
	jsonData.Overview = overview
	for _, s := range summaries {
		commentRatio := 0.0
		if s.Lines > 0 {
			commentRatio = float64(s.CommentLines) / float64(s.Lines) * 100
		}
		maxFuncLines := 0
		for _, f := range s.Functions {
			if f.LineCount > maxFuncLines {
				maxFuncLines = f.LineCount
			}
		}
		jsonData.Files = append(jsonData.Files, JSONSummary{
			Filename:           s.Filename,
			Package:            s.Package,
			Types:              s.Types,
			Functions:          s.Functions,
			Imports:            s.Imports,
			Lines:              s.Lines,
			CommentLines:       s.CommentLines,
			MaxFuncLines:       maxFuncLines,
			CommentRatio:       commentRatio,
			LongFunctions:      s.LongFunctions,
			AvgComplexity:      s.AvgComplexity,
			GodocCoverage:      s.GodocCoverage,
			MaxFunctionDepth:   s.MaxFunctionDepth,
			MaintainabilityIdx: s.MaintainabilityIdx,
		})
	}

	data, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	return os.WriteFile(outputPath, data, 0644)
}

func main() {
	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}

	goFiles, err := scanDirectory(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var summaries []CodeSummary
	for _, file := range goFiles {
		summary, err := parseFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			continue
		}
		summaries = append(summaries, summary)
	}

	if len(summaries) == 0 {
		fmt.Println("No Go files found.")
		return
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Filename < summaries[j].Filename
	})

	var errors []error
	if err := generateMarkdown(summaries, "go_code_summary.md"); err != nil {
		errors = append(errors, fmt.Errorf("generating Markdown: %w", err))
	}
	if err := generateHTML(summaries, "go_code_summary.html"); err != nil {
		errors = append(errors, fmt.Errorf("generating HTML: %w", err))
	}
	if err := generateJSON(summaries, "go_code_summary.json"); err != nil {
		errors = append(errors, fmt.Errorf("generating JSON: %w", err))
	}

	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("Generated go_code_summary.md, go_code_summary.html, and go_code_summary.json")
}
