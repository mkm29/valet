package tests

import (
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
}

func TestValet(t *testing.T) {
	suite.Run(t, new(ValetTestSuite))
}
