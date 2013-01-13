package harness

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

const validationKeysSource = `
package test

func (c *Application) testFunc(a, b int, user models.User) rev.Result {
	// Line 5
	c.Validation.Required(a)
	c.Validation.Required(a).Message("Error message")
	c.Validation.Required(a).
		Message("Error message")

	// Line 11
	c.Validation.Required(user.Name)
	c.Validation.Required(user.Name).Message("Error message")

	// Line 15
	c.Validation.MinSize(b, 12)
	c.Validation.MinSize(b, 12).Message("Error message")
	c.Validation.MinSize(b,
		12)

	// Line 21
	c.Validation.Required(b == 5)
}

func (m Model) Validate(v *rev.Validation) {
	// Line 26
	v.Required(m.name)
	v.Required(m.name == "something").
		Message("Error Message")
}
`

var expectedValidationKeys = []map[int]string{
	{
		6:  "a",
		7:  "a",
		8:  "a",
		12: "user.Name",
		13: "user.Name",
		16: "b",
		17: "b",
		19: "b",
		22: "b",
	}, {
		27: "m.name",
		28: "m.name",
	},
}

// This tests the recording of line number to validation key of the preceeding
// example source.
func TestGetValidationKeys(t *testing.T) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "validationKeysSource", validationKeysSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Decls) != 2 {
		t.Fatal("Expected 2 decl in the source, found", len(file.Decls))
	}

	for i, decl := range file.Decls {
		lineKeys := getValidationKeys(fset, decl.(*ast.FuncDecl))
		for k, v := range expectedValidationKeys[i] {
			if lineKeys[k] != v {
				t.Errorf("Not found - %d: %v - Actual Map: %v", k, v, lineKeys)
			}
		}

		if len(lineKeys) != len(expectedValidationKeys[i]) {
			t.Error("Validation key map not the same size as expected:", lineKeys)
		}
	}
}

var TypeExprs = map[string]TypeExpr{
	"int":        TypeExpr{"int", "", 0},
	"*int":       TypeExpr{"*int", "", 1},
	"[]int":      TypeExpr{"[]int", "", 2},
	"...int":     TypeExpr{"[]int", "", 2},
	"[]*int":     TypeExpr{"[]*int", "", 3},
	"...*int":    TypeExpr{"[]*int", "", 3},
	"MyType":     TypeExpr{"MyType", "pkg", 0},
	"*MyType":    TypeExpr{"*MyType", "pkg", 1},
	"[]MyType":   TypeExpr{"[]MyType", "pkg", 2},
	"...MyType":  TypeExpr{"[]MyType", "pkg", 2},
	"[]*MyType":  TypeExpr{"[]*MyType", "pkg", 3},
	"...*MyType": TypeExpr{"[]*MyType", "pkg", 3},
}

func TestTypeExpr(t *testing.T) {
	for typeStr, expected := range TypeExprs {
		// Handle arrays and ... myself, since ParseExpr() does not.
		array := strings.HasPrefix(typeStr, "[]")
		if array {
			typeStr = typeStr[2:]
		}

		ellipsis := strings.HasPrefix(typeStr, "...")
		if ellipsis {
			typeStr = typeStr[3:]
		}

		expr, err := parser.ParseExpr(typeStr)
		if err != nil {
			t.Error("Failed to parse test expr:", typeStr)
			continue
		}

		if array {
			expr = &ast.ArrayType{expr.Pos(), nil, expr}
		}
		if ellipsis {
			expr = &ast.Ellipsis{expr.Pos(), expr}
		}

		actual := NewTypeExpr("pkg", expr)
		if !reflect.DeepEqual(expected, actual) {
			t.Error("Fail, expected", expected, ", was", actual)
		}
	}
}
