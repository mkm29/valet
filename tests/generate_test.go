package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mkm29/valet/cmd"
)

func (ts *ValetTestSuite) TestNewGenerateCmd() {
	app := cmd.NewApp()
	generateCmd := cmd.NewGenerateCmdWithApp(app)
	ts.Equal("generate [context-dir]", generateCmd.Use, "Command use should be 'generate [context-dir]'")
	ts.Equal("Generate JSON Schema from values.yaml", generateCmd.Short, "unexpected Short description")
	ts.NotNil(generateCmd.Args, "expected Args validator to be set")
}

// TestGenerateCmd_OverrideFileNotFound ensures error if overrides flag points to non-existent file
func (ts *ValetTestSuite) TestGenerateCmd_OverrideFileNotFound() {
	tmp := ts.T().TempDir()
	// Create values.yaml to pass values check
	os.WriteFile(filepath.Join(tmp, "values.yaml"), []byte("x: 1\n"), 0644)
	app := cmd.NewApp()
	generateCmd := cmd.NewGenerateCmdWithApp(app)
	generateCmd.SetOut(new(bytes.Buffer))
	generateCmd.SetErr(new(bytes.Buffer))
	// Set overrides flag to non-existent file
	generateCmd.SetArgs([]string{"--overrides", "nofile.yaml", tmp})
	err := generateCmd.Execute()
	ts.Error(err)
	ts.Contains(err.Error(), "overrides file nofile.yaml not found")
}

// TestGenerateCmd_NoValues tests generate fails when no values file present
func (ts *ValetTestSuite) TestGenerateCmd_NoValues() {
	tmp := ts.T().TempDir()
	app := cmd.NewApp()
	generateCmd := cmd.NewGenerateCmdWithApp(app)
	// Capture error from Execute
	generateCmd.SetArgs([]string{tmp})
	err := generateCmd.Execute()
	ts.Error(err)
	ts.Contains(err.Error(), "no values.yaml or values.yml found in")
}

// Test basic schema generation without overrides
func (ts *ValetTestSuite) TestGenerate_Simple() {
	tmp := ts.T().TempDir()
	// Create values.yaml
	yaml := []byte(
		"foo: bar\n" +
			"num: 42\n" +
			"flag: true\n",
	)
	err := os.WriteFile(filepath.Join(tmp, "values.yaml"), yaml, 0644)
	ts.Require().NoError(err, "failed to write values.yaml")

	// Run Generate
	app := cmd.NewApp()
	msg, err := cmd.GenerateWithApp(app, tmp, "")
	ts.Require().NoError(err, "Generate failed")

	// Expect message about generation
	expectedMsg := filepath.Join(tmp, "values.schema.json")
	ts.Equal("Generated "+expectedMsg+" from values.yaml", msg)

	// Read and unmarshal schema
	data, err := os.ReadFile(filepath.Join(tmp, "values.schema.json"))
	ts.Require().NoError(err, "failed to read schema")

	var schema map[string]interface{}
	err = json.Unmarshal(data, &schema)
	ts.Require().NoError(err, "invalid JSON schema")

	// Basic checks
	ts.Equal("object", schema["type"], "expected type object")

	props, ok := schema["properties"].(map[string]interface{})
	ts.Require().True(ok, "properties missing or wrong type")

	// Check foo default
	foo, ok := props["foo"].(map[string]interface{})
	ts.Require().True(ok, "foo property missing")
	ts.Equal("bar", foo["default"], "foo default incorrect")

	// Check num default
	num, ok := props["num"].(map[string]interface{})
	ts.Require().True(ok, "num property missing")
	ts.Equal(float64(42), num["default"], "num default incorrect")

	// Check flag default
	flagp, ok := props["flag"].(map[string]interface{})
	ts.Require().True(ok, "flag property missing")
	ts.Equal(true, flagp["default"], "flag default incorrect")
}

// TestGenerateCommand_Execute runs the generate subcommand end-to-end
func (ts *ValetTestSuite) TestGenerateCommand_Execute() {
	tmp := ts.T().TempDir()
	// Create values.yaml
	yaml := []byte("a: alpha\nb: beta\n")
	err := os.WriteFile(filepath.Join(tmp, "values.yaml"), yaml, 0644)
	ts.Require().NoError(err, "write values.yaml failed")

	app := cmd.NewApp()
	generateCmd := cmd.NewGenerateCmdWithApp(app)
	// Use absolute path to temp dir
	generateCmd.SetArgs([]string{tmp})
	err = generateCmd.Execute()
	ts.Require().NoError(err, "GenerateCmd.Execute failed")

	// Check file exists at expected location
	outFile := filepath.Join(tmp, "values.schema.json")
	_, err = os.Stat(outFile)
	ts.NoError(err, "expected schema file at %s", outFile)
}

// TestGenerateCmd_MissingArg ensures subcommand errors on missing context arg
func (ts *ValetTestSuite) TestGenerateCmd_MissingArg() {
	app := cmd.NewApp()
	generateCmd := cmd.NewGenerateCmdWithApp(app)
	generateCmd.SetOut(new(bytes.Buffer))
	generateCmd.SetErr(new(bytes.Buffer))
	generateCmd.SetArgs([]string{})
	err := generateCmd.Execute()
	ts.Error(err, "expected error when missing context argument")
}

// TestGenerateCmd_Help ensures help text is shown without error
func (ts *ValetTestSuite) TestGenerateCmd_Help() {
	app := cmd.NewApp()
	generateCmd := cmd.NewGenerateCmdWithApp(app)
	var out bytes.Buffer
	generateCmd.SetOut(&out)
	generateCmd.SetErr(&out)
	generateCmd.SetArgs([]string{"-h"})
	err := generateCmd.Execute()
	ts.NoError(err, "expected help to succeed")
	ts.Contains(out.String(), "Generate JSON Schema", "unexpected help output")
}

// Test schema generation with overrides
func (ts *ValetTestSuite) TestGenerate_Override() {
	tmp := ts.T().TempDir()
	// Create values.yaml
	yaml1 := []byte("a: 1\nes: test\n")
	err := os.WriteFile(filepath.Join(tmp, "values.yaml"), yaml1, 0644)
	ts.Require().NoError(err, "failed to write values.yaml")

	// Create overrides.yaml
	yaml2 := []byte("a: 2\nb: new\n")
	err = os.WriteFile(filepath.Join(tmp, "over.yaml"), yaml2, 0644)
	ts.Require().NoError(err, "failed to write overrides")

	app := cmd.NewApp()
	msg, err := cmd.GenerateWithApp(app, tmp, "over.yaml")
	ts.Require().NoError(err, "Generate failed")

	expectedMsg := filepath.Join(tmp, "values.schema.json")
	ts.Equal("Generated "+expectedMsg+" by merging over.yaml into values.yaml", msg)

	// Read schema
	data, err := os.ReadFile(filepath.Join(tmp, "values.schema.json"))
	ts.Require().NoError(err, "failed to read schema")

	var schema map[string]interface{}
	err = json.Unmarshal(data, &schema)
	ts.Require().NoError(err, "invalid JSON schema")

	props := schema["properties"].(map[string]interface{})
	// a should be default 2
	a := props["a"].(map[string]interface{})
	ts.Equal(float64(2), a["default"], "override a default incorrect")

	// b should appear
	b := props["b"].(map[string]interface{})
	ts.Equal("new", b["default"], "override b default incorrect")
}

// TestGenerate_EmptyValues tests the Generate function with empty values
func (ts *ValetTestSuite) TestGenerate_EmptyValues() {
	tmp := ts.T().TempDir()

	// Create empty values.yaml
	emptyYaml := []byte("{}\n")
	err := os.WriteFile(filepath.Join(tmp, "values.yaml"), emptyYaml, 0644)
	ts.Require().NoError(err, "failed to write values.yaml")

	// Run Generate - don't check the message since it's already tested elsewhere
	app := cmd.NewApp()
	_, err = cmd.GenerateWithApp(app, tmp, "")
	ts.Require().NoError(err, "Generate failed")

	// Read schema and check
	data, err := os.ReadFile(filepath.Join(tmp, "values.schema.json"))
	ts.Require().NoError(err, "failed to read schema")

	var schema map[string]interface{}
	err = json.Unmarshal(data, &schema)
	ts.Require().NoError(err, "invalid JSON schema")

	// Basic checks for empty schema
	ts.Equal("object", schema["type"], "expected type object")

	props, ok := schema["properties"].(map[string]interface{})
	ts.True(ok && len(props) == 0, "properties should be empty")
}

// TestGenerate_MissingValuesYaml tests Generate with values.yml instead of values.yaml
func (ts *ValetTestSuite) TestGenerate_ValuesYml() {
	tmp := ts.T().TempDir()

	// Create values.yml instead of values.yaml
	yaml := []byte("foo: bar\n")
	err := os.WriteFile(filepath.Join(tmp, "values.yml"), yaml, 0644)
	ts.Require().NoError(err, "failed to write values.yml")

	// Run Generate
	app := cmd.NewApp()
	_, err = cmd.GenerateWithApp(app, tmp, "")
	ts.Require().NoError(err, "Generate failed")

	// Check schema was created
	schemaPath := filepath.Join(tmp, "values.schema.json")
	_, err = os.Stat(schemaPath)
	ts.NoError(err, "schema file not created")
}

// TestGenerate_InvalidYAML tests Generate with invalid YAML
func (ts *ValetTestSuite) TestGenerate_InvalidYAML() {
	tmp := ts.T().TempDir()

	// Create invalid YAML - use syntax that will actually fail YAML parsing
	invalidYaml := []byte("foo: [bar: baz}\n")
	err := os.WriteFile(filepath.Join(tmp, "values.yaml"), invalidYaml, 0644)
	ts.Require().NoError(err, "failed to write values.yaml")

	// Run Generate - expect error
	app := cmd.NewApp()
	_, err = cmd.GenerateWithApp(app, tmp, "")
	ts.Error(err)
	ts.Contains(err.Error(), "error", "expected error for invalid YAML")
}

// TestGenerate_InvalidOverrides tests Generate with invalid overrides path
func (ts *ValetTestSuite) TestGenerate_InvalidOverrides() {
	tmp := ts.T().TempDir()

	// Create values.yaml
	yaml := []byte("foo: bar\n")
	err := os.WriteFile(filepath.Join(tmp, "values.yaml"), yaml, 0644)
	ts.Require().NoError(err, "failed to write values.yaml")

	// Create invalid overrides.yaml
	invalidYaml := []byte("foo: [bar: baz}\n")
	err = os.WriteFile(filepath.Join(tmp, "overrides.yaml"), invalidYaml, 0644)
	ts.Require().NoError(err, "failed to write overrides.yaml")

	// Run Generate - expect error
	app := cmd.NewApp()
	_, err = cmd.GenerateWithApp(app, tmp, "overrides.yaml")
	ts.Error(err)
	ts.Contains(err.Error(), "error", "expected error for invalid overrides")
}
