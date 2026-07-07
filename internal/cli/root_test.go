package cli_test

import (
	"bytes"
	"testing"

	"github.com/andespath/andes-ai/internal/cli"
)

func TestRootCmdShowsHelp(t *testing.T) {
	root := cli.NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("andes")) {
		t.Errorf("help no menciona 'andes':\n%s", out.String())
	}
}
