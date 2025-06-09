package tests

import "github.com/mkm29/valet/cmd"

func (ts *ValetTestSuite) TestNewRootCmd() {
	cmd := cmd.NewRootCmd()
	ts.NotNil(cmd, "NewRootCmd should not return nil")
	ts.Equal("valet", cmd.Use, "Command use should be 'valet'")
}
