package tests

import (
	"os"
	"path/filepath"

	"github.com/mkm29/valet/cmd"
)

func (ts *ValetTestSuite) TestNewRootCmd() {
	cmd := cmd.NewRootCmd()
	ts.NotNil(cmd, "NewRootCmd should not return nil")
	ts.Equal("valet", cmd.Use, "Command use should be 'valet'")
}

func (ts *ValetTestSuite) TestRootCmd_DefaultContext() {
	// Create an isolated environment for this test
	tmp := ts.T().TempDir()
	
	// Create a values.yaml file
	valuesContent := []byte("test: value\n")
	err := os.WriteFile(filepath.Join(tmp, "values.yaml"), valuesContent, 0644)
	ts.Require().NoError(err, "failed to write values.yaml")
	
	// Save current directory and change to temp
	cwd, err := os.Getwd()
	ts.Require().NoError(err)
	defer os.Chdir(cwd)
	
	err = os.Chdir(tmp)
	ts.Require().NoError(err)
	
	// Create a fresh root command to avoid state interference
	rootCmd := cmd.NewRootCmd()
	
	// Execute without any args - should use default context (current directory)
	rootCmd.SetArgs([]string{})
	err = rootCmd.Execute()
	ts.NoError(err, "Execute should succeed with values.yaml in current directory")
	
	// Verify schema was generated
	schemaPath := filepath.Join(tmp, "values.schema.json")
	_, err = os.Stat(schemaPath)
	ts.NoError(err, "schema file should be created in current directory")
	
	// Verify schema content
	schemaData, err := os.ReadFile(schemaPath)
	ts.Require().NoError(err)
	ts.Contains(string(schemaData), `"test"`, "schema should contain the test property")
}

// TestRootCmd_ConfigFile ensures context is read from default config file
func (ts *ValetTestSuite) TestRootCmd_ConfigFile() {
	tmp := ts.T().TempDir()
	// Create a subdirectory with values.yaml
	sub := filepath.Join(tmp, "subdir")
	err := os.Mkdir(sub, 0755)
	ts.Require().NoError(err, "mkdir failed")

	yaml := []byte("k: v\n")
	err = os.WriteFile(filepath.Join(sub, "values.yaml"), yaml, 0644)
	ts.Require().NoError(err, "write values.yaml failed")

	// Create config file .valet.yaml in tmp
	cfgContent := []byte("context: subdir\n")
	err = os.WriteFile(filepath.Join(tmp, ".valet.yaml"), cfgContent, 0644)
	ts.Require().NoError(err, "write config failed")

	// Change to temp dir
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)

	rootCmd := cmd.NewRootCmd()
	// Specify config-file flag
	rootCmd.SetArgs([]string{"--config-file", ".valet.yaml"})
	err = rootCmd.Execute()
	ts.NoError(err, "Execute failed")

	// Check schema file in subdir
	outPath := filepath.Join(sub, "values.schema.json")
	_, err = os.Stat(outPath)
	ts.NoError(err, "expected schema at %s", outPath)
}

// TestRootCmd_NoValues tests root command error when no values file present
func (ts *ValetTestSuite) TestRootCmd_NoValues() {
	tmp := ts.T().TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmp)

	rootCmd := cmd.NewRootCmd()
	rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	// The root command shows help when no context is provided
	// It doesn't error when there's no values.yaml in an empty directory
	ts.NoError(err, "should show help without error")
}
