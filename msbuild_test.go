package dotnet

import (
	"slices"
	"testing"
)

func TestDiagnosticsReadsMSBuildErrorsOnce(t *testing.T) {
	out := `  Plugin.cs(12,5): error CS1002: ; expected [/p/src/A/A.csproj]
/p/src/A/Plugin.cs(3,1,3,9): warning CS0168: The variable 'x' is declared but never used [/p/src/A/A.csproj]
  Plugin.cs(12,5): error CS1002: ; expected [/p/src/A/A.csproj]
Build FAILED.
`
	got := Diagnostics(out)
	want := []Diagnostic{
		{File: "Plugin.cs", Line: 12, Column: 5, Severity: "error", Code: "CS1002", Message: "; expected"},
		{File: "/p/src/A/Plugin.cs", Line: 3, Column: 1, Severity: "warning", Code: "CS0168", Message: "The variable 'x' is declared but never used"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("diagnostics = %+v", got)
	}
}
