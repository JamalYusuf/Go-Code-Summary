# 📝 Go Code Summarizer

**Go Code Summarizer** is a powerful tool for analyzing Go projects, providing detailed insights into code structure, quality, and maintainability. It recursively scans a Go project directory, parses `.go` files (excluding tests), and generates three output files: a Markdown report, an HTML dashboard, and a JSON summary. Designed for dev leads and developers, it offers metrics like cyclomatic complexity, godoc coverage, and project health to support architectural decisions and code reviews.

## 🚀 Features

- **Recursive Scanning**: Processes all `.go` files in a directory, skipping `_test.go` files.
- **AST Parsing**: Uses `go/parser` and `go/ast` to extract:
  - Type declarations (structs, interfaces) with comments.
  - Function declarations (including methods) with signatures and comments.
  - Both exported and unexported identifiers.
- **Comprehensive Metrics**:
  - 📏 Lines of code and comment-to-code ratio.
  - 🛠️ Function count and long functions (>50 lines).
  - 🧠 Cyclomatic complexity per function and file.
  - 📖 Godoc coverage for exported identifiers.
  - 🔲 Maximum function nesting depth.
  - 🛡️ Maintainability index per file.
  - 📦 Package breakdown (files, lines, imports, coupling).
  - 🔗 External dependency count.
  - 🏥 Project health score (0–100).
  - 🚨 Risky file detection (high complexity, low documentation).
  - ⏰ Refactoring effort estimate (person-hours).
- **Output Formats**:
  - **Markdown** (`go_code_summary.md`): Readable report with emojis, tables, and code blocks.
  - **HTML** (`go_code_summary.html`): Interactive dashboard with TailwindCSS styling and Chart.js visualizations.
  - **JSON** (`go_code_summary.json`): Machine-readable data for integration with CI/CD or analytics tools.
- **Visual Design**: Emojis (📝, 📊, 📂) enhance readability in Markdown and HTML outputs.
- **Robustness**: Handles edge cases (empty directories, no exports, malformed files) with clear error messages.

## 📦 Installation

### Prerequisites

- **Go**: Version 1.16 or later (uses standard library only, no external dependencies).
- A Go project directory to analyze.

### Steps

1. **Clone the Repository**:

   ```bash
   git clone https://github.com/JamalYusuf/Go-Code-Summary.git
   cd go-code-summarizer
   ```

2. **Verify Go Installation**:

   ```bash
   go version
   ```

   Ensure output shows Go 1.16 or higher.

3. **No Build Required**: The program runs directly with `go run`.

## 🛠️ Usage

Run the summarizer on a Go project directory:

```bash
go run summarize.go [path/to/directory]
```

- If no directory is specified, it defaults to the current directory (`.`).
- The program generates three files in the working directory:
  - `go_code_summary.md`
  - `go_code_summary.html`
  - `go_code_summary.json`

### Example

To analyze a project in `~/my-go-project`:

```bash
go run summarize.go ~/my-go-project
```

Output:
```
Generated go_code_summary.md, go_code_summary.html, and go_code_summary.json
```

- Open `go_code_summary.html` in a browser for an interactive dashboard.
- View `go_code_summary.md` in a Markdown viewer or GitHub.
- Parse `go_code_summary.json` for automated workflows.

## 📊 Output Details

### Markdown (`go_code_summary.md`)

- **Project Overview**: Summarizes total files, lines, functions, and advanced metrics.
- **Package Breakdown**: Table of packages with file counts, lines, imports, and coupling.
- **Per-File Details**:
  - Metrics (lines, functions, complexity, etc.).
  - Types and functions with comments and code blocks.
- Uses emojis (e.g., 📂, 🛠️) for clarity.

**Example Snippet**:

```markdown
📝 # Go Code Summary

📊 ## Project Overview

- 📂 Files Processed: 2
- 📏 Total Lines of Code: 150
- 🛠️ Total Functions: 10
- ⚠️ Long Functions (>50 lines): 1
- 📜 Average Comment-to-Code Ratio: 25.00%
- 🧠 Average Function Complexity: 2.50
- 📖 Godoc Coverage: 80.00%

📂 ## main.go (main)

📈 **Metrics**:
- 📏 Lines of Code: 100
- 🛠️ Number of Functions: 6
- 📖 Godoc Coverage: 75.00%
```

### HTML (`go_code_summary.html`)

- **Dashboard**: Collapsible sections per file, styled with TailwindCSS.
- **Visualizations**: Bar chart of package file and line counts (via Chart.js).
- **Metrics**: Same as Markdown, with emojis and dark-themed code blocks.
- **Interactive**: Expand/collapse files using `<details>` tags.

**Note**: Requires an internet connection to load TailwindCSS and Chart.js CDNs.

### JSON (`go_code_summary.json`)

- **Structure**:
  - `overview`: Project-wide metrics (files, lines, health score, etc.).
  - `files`: Array of per-file summaries (types, functions, metrics).
- Ideal for CI/CD integration or custom analysis.

**Example Snippet**:

```json
{
  "overview": {
    "total_files": 2,
    "total_lines": 150,
    "project_health": 85.5
  },
  "files": [
    {
      "filename": "main.go",
      "package": "main",
      "lines": 100,
      "godoc_coverage": 75
    }
  ]
}
```

## 📈 Metrics Explained

- **Lines of Code**: Total lines per file and project.
- **Comment-to-Code Ratio**: Percentage of comment lines, indicating documentation effort.
- **Long Functions**: Functions >50 lines, flagged for potential refactoring (per Go best practices).
- **Cyclomatic Complexity**: Counts control flow paths (if, for, switch, etc.) per function, averaged per file.
- **Godoc Coverage**: Percentage of exported identifiers (types, functions) with comments.
- **Function Depth**: Maximum nesting level in functions, highlighting complexity.
- **Maintainability Index**: Score (0–100) balancing lines, complexity, and comments.
- **Package Coupling**: Number of internal package dependencies, showing modularity.
- **Project Health Score**: Weighted score (0–100) based on comments (30%), godoc (30%), long functions (20%), and complexity (20%).
- **Risky Files**: Files with high complexity (>5), low godoc (<50%), or many long functions (>3).
- **Effort Estimate**: Person-hours for refactoring, based on lines (0.5h/100), complexity (0.2h/point), and long functions (5h each).

## 🐛 Error Handling

- **Invalid Files**: Skips unparsable `.go` files with a warning.
- **Empty Directories**: Outputs a message and exits cleanly.
- **Edge Cases**: Handles zero lines, no exports, or empty functions gracefully.
- **Errors**: Reported to stderr with context (e.g., "parsing file main.go: syntax error").

## 🤝 Contributing

Contributions are welcome! To contribute:

1. **Fork the Repository**.
2. **Create a Branch**:

   ```bash
   git checkout -b feature/your-feature
   ```

3. **Make Changes**:
   - Follow Go coding standards (e.g., `gofmt`, clear comments).
   - Update tests if adding features (add tests to `summarize_test.go` when implemented).
   - Keep outputs consistent (Markdown, HTML, JSON).

4. **Test Locally**:

   ```bash
   go run summarize.go ./testdata
   ```

5. **Submit a Pull Request**:
   - Describe changes clearly.
   - Reference any issues fixed.

### Ideas for Contributions

- Add more metrics (e.g., test coverage, interface usage).
- Enhance HTML visualizations (e.g., complexity graphs).
- Support custom output formats (e.g., CSV).
- Improve performance for large projects (e.g., parallel parsing).

## 📜 License

MIT License. See [LICENSE](LICENSE) for details.

## 🙌 Acknowledgments

- Built with the Go standard library (`go/parser`, `go/ast`).
- Styled with [TailwindCSS](https://tailwindcss.com) and [Chart.js](https://www.chartjs.org).
- Inspired by Go’s philosophy of simplicity and clarity.
