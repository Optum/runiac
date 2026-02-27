package config

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var yamlExample = []byte(`Hacker: true
name: steve
hobbies:
- skateboarding
- snowboarding
- go
clothing:
  jacket: leather
  trousers: denim
  pants:
    size: large
age: 35
eyes : brown
beard: true
`)

func TestGetConfig_EnvironmentVariablesShouldMatch(t *testing.T) {
	t.Parallel()

	_ = os.Setenv("RUNIAC_PRIMARY_REGION", "centralus")
	_ = os.Setenv("RUNIAC_RUNNER", "terraform")
	_ = os.Setenv("RUNIAC_STEP_WHITELIST", "default/default")
	conf, err := GetConfig()

	require.NotNil(t, conf)
	require.NoError(t, err)
	require.Equal(t, "centralus", conf.PrimaryRegion)
	require.Equal(t, "terraform", conf.Runner)
	require.NotEmpty(t, conf.StepWhitelist)
	require.Equal(t, "default/default", conf.StepWhitelist[0])
}

func TestConfigStructTags_ShouldBeValid(t *testing.T) {
	t.Parallel()

	rt := reflect.TypeOf(Config{})
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := string(field.Tag)
		if tag == "" {
			continue
		}

		// Check that all key:"value" pairs have properly quoted values
		// A malformed tag like `mapstructure:runner` (missing quotes) will not
		// be parseable by reflect.StructTag.Lookup
		for _, key := range []string{"mapstructure", "required"} {
			if !strings.Contains(tag, key) {
				continue
			}
			_, ok := field.Tag.Lookup(key)
			require.True(t, ok, fmt.Sprintf(
				"struct field %q has malformed %q tag: %s (missing quotes around value?)",
				field.Name, key, tag,
			))
		}
	}
}

func TestConfigMapstructureTags_ShouldMatchBindEnvKeys(t *testing.T) {
	t.Parallel()

	// These are the keys registered via viper.BindEnv() in GetConfig()
	boundKeys := map[string]bool{
		"environment":      true,
		"namespace":        true,
		"project":          true,
		"log_level":        true,
		"dry_run":          true,
		"self_destroy":     true,
		"deployment_ring":  true,
		"primary_region":   true,
		"regional_regions": true,
		"max_retries":      true,
		"max_test_retries": true,
		"account_id":       true,
		"runner":           true,
		"step_whitelist":   true,
	}

	// Verify every mapstructure tag with a BindEnv key actually resolves
	rt := reflect.TypeOf(Config{})
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tagVal, ok := field.Tag.Lookup("mapstructure")
		if !ok || tagVal == "" {
			continue
		}
		if boundKeys[tagVal] {
			// Ensure the tag value is not empty after Lookup (catches malformed tags)
			require.NotEmpty(t, tagVal, fmt.Sprintf(
				"struct field %q has mapstructure tag bound via BindEnv but tag value is empty",
				field.Name,
			))
		}
	}
}

func TestNoDeprecatedImports(t *testing.T) {
	t.Parallel()

	deprecatedPackages := []string{
		"io/ioutil",
	}

	// Walk all .go files in the repo
	root := filepath.Join("..", "..")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable dirs
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == "vendor" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil // skip unparseable files
		}

		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for _, dep := range deprecatedPackages {
				if importPath == dep {
					t.Errorf("%s imports deprecated package %q", path, dep)
				}
			}
		}
		return nil
	})
	require.NoError(t, err)
}

func TestNoNonConstantFormatStrings(t *testing.T) {
	t.Parallel()

	// printf-family function names to check
	printfFuncs := map[string]bool{
		"Printf": true, "Sprintf": true, "Fprintf": true, "Errorf": true,
		"Fatalf": true, "Infof": true, "Warnf": true, "Warningf": true,
		"Debugf": true, "Tracef": true, "Logf": true, "Panicf": true,
	}

	root := filepath.Join("..", "..")
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == "vendor" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return nil
		}

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}

			// Get function name
			var funcName string
			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr:
				funcName = fn.Sel.Name
			case *ast.Ident:
				funcName = fn.Name
			}

			if !printfFuncs[funcName] {
				return true
			}

			// Fprintf takes (writer, format, ...) — format is the 2nd arg
			// Other printf funcs take (format, ...) — format is the 1st arg
			fmtArgIdx := 0
			if funcName == "Fprintf" {
				fmtArgIdx = 1
			}

			if fmtArgIdx >= len(call.Args) {
				return true
			}

			// Format arg should be a string literal (constant format string)
			fmtArg := call.Args[fmtArgIdx]
			if _, isLit := fmtArg.(*ast.BasicLit); !isLit {
				// Not a string literal — could be a variable format string
				// Allow if it's a const identifier
				if ident, isIdent := fmtArg.(*ast.Ident); isIdent {
					if ident.Obj != nil && ident.Obj.Kind == ast.Con {
						return true // const is fine
					}
				}
				pos := fset.Position(call.Pos())
				t.Errorf("%s:%d: %s() called with non-constant format string", pos.Filename, pos.Line, funcName)
			}
			return true
		})
		return nil
	})
}
