//go:build integration

package cmd

import "bytes"

func runCmd(args ...string) (*bytes.Buffer, error) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	return buf, cmd.Execute()
}
