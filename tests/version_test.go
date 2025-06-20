package tests

import "github.com/mkm29/valet/cmd"

func (ts *ValetTestSuite) TestNewVersionCmd() {
	cmd := cmd.NewVersionCmd()
	ts.Equal("version", cmd.Use, "expected Use 'version'")
	ts.Equal("Print version information", cmd.Short, "expected Short 'Print version information'")
	ts.NotEmpty(cmd.Long, "expected non-empty Long description")
	ts.NotNil(cmd.Run, "expected Run function to be set")
}

// Note: The original version_test.go contained more comprehensive tests for showVersion,
// but since that function is unexported, we can only test the public API.
// The version functionality is still covered through integration testing.
