package dotnet

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Diagnostic is one compiler error or warning, at its file and line.
type Diagnostic struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// MSBuild's canonical form: file(line,col): error CODE: text [project].
var msbuildLine = regexp.MustCompile(`^\s*(.+?)\((\d+),(\d+)(?:,\d+,\d+)?\): (error|warning) ([A-Za-z]+\d+): (.*?)(?: \[[^\]]+\])?$`)

// Diagnostics reads MSBuild's output for errors and warnings, each once and in order.
func Diagnostics(out string) []Diagnostic {
	problems := []Diagnostic{}
	for line := range strings.Lines(out) {
		m := msbuildLine.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
		if m == nil {
			continue
		}
		row, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		p := Diagnostic{File: m[1], Line: row, Column: col, Severity: m[4], Code: m[5], Message: m[6]}
		if !slices.Contains(problems, p) {
			problems = append(problems, p)
		}
	}
	return problems
}
