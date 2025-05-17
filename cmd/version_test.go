package cmd

import (
   "bytes"
   "io"
   "os"
   "testing"

   "debug/buildinfo"
   debugpkg "runtime/debug"
)

func TestShowVersion_Success(t *testing.T) {
   // Backup and restore
   oldExit := exit
   oldRead := readBuildInfo
   oldStdout := os.Stdout
   r, w, _ := os.Pipe()
   os.Stdout = w
   defer func() {
       exit = oldExit
       readBuildInfo = oldRead
       os.Stdout = oldStdout
   }()
   // Stub build info
   readBuildInfo = func(path string) (*buildinfo.BuildInfo, error) {
       return &buildinfo.BuildInfo{
           Main: debugpkg.Module{Path: "mod/path", Version: "vX.Y.Z"},
           Settings: []debugpkg.BuildSetting{{Key: "vcs.revision", Value: "abcdef"}},
       }, nil
   }
   // Capture exit code via panic
   exit = func(code int) { panic(code) }
   // Call showVersion and capture output
   var output bytes.Buffer
   done := make(chan struct{})
   go func() {
       defer func() {
           if rec := recover(); rec != nil {
               // Expect exit(0)
               if code, ok := rec.(int); !ok || code != 0 {
                   t.Errorf("unexpected exit code: %v", rec)
               }
           } else {
               t.Error("expected exit(0) panic")
           }
           w.Close()
           io.Copy(&output, r)
           close(done)
       }()
       showVersion()
   }()
   <-done
   // Verify output
   expected := "mod/path@vX.Y.Z (commit abcdef)\n"
   if output.String() != expected {
       t.Errorf("expected %q, got %q", expected, output.String())
   }
}