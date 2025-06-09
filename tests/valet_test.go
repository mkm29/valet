package tests

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ValetTestSuite struct {
	suite.Suite
}

func (ts *ValetTestSuite) CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

// SetupSuite is called before the suite starts running
func (suite *ValetTestSuite) SetupSuite() {
	suite.T().Log("Setting up Valet test suite")

	var cwd, _ = os.Getwd()

	// 1. Create a test directory
	tempDir := suite.T().TempDir()
	suite.T().Logf("Using temporary directory: %s", tempDir)

	configContent := fmt.Sprintf(`context: %s
overrides: %s
output: %s
debug: %t
`, tempDir, filepath.Join(tempDir, "overrides.yaml"), filepath.Join(tempDir, "values.schema.json"), true)

	// 2. Create a .valet.yaml file in the test directory
	valetConfig := []byte(configContent)
	if err := os.WriteFile(filepath.Join(tempDir, ".valet.yaml"), valetConfig, 0644); err != nil {
		suite.T().Fatalf("Failed to create .valet.yaml file: %v", err)
	}
	// 3. Create a sample Helm chart to use for testing
	chartDir := filepath.Join(tempDir, "mychart")
	if err := os.MkdirAll(chartDir, 0755); err != nil {
		suite.T().Fatalf("Failed to create chart directory: %v", err)
	}
	// 4. Copy mychart from ./testdata to the test directory
	if err := suite.CopyDir(filepath.Join(cwd, "..", "testdata", "mychart"), chartDir); err != nil {
		suite.T().Fatalf("Failed to copy mychart: %v", err)
	}

}

func TestValet(t *testing.T) {
	suite.Run(t, new(ValetTestSuite))
}
