//go:build windows

package notify

import "testing"

func TestRunHasEntrypointSignature(t *testing.T) {
	var run func(ToolConfig, []string) int = Run
	if run == nil {
		t.Fatal("Run should be available as the shared CLI entrypoint")
	}
}
