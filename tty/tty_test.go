package tty

import (
	"context"
	"testing"
	"time"
)

// TODO use testify and split into separate tests
func TestRunCommandWithCancel(t *testing.T) {
	testcases := []struct {
		name     string
		init     string
		cmds     []string
		args     []string
		expected int
		valid    bool
	}{
		{
			name:     "non-shell command",
			init:     "pwd",
			cmds:     []string{"pwd\n", "pwd\n"},
			expected: 1,
			valid:    true,
		},
		{
			name:     "bash",
			init:     "bash",
			args:     []string{"-i"},
			cmds:     []string{"pwd\n", "pwd\n", "pwd\n", "pwd\n", "pwd\n"},
			expected: 11,
			valid:    true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cncl := context.WithCancel(context.Background())
			term, err := NewTerminal(ctx, tc.init, tc.args...)
			if err != nil {
				t.Fatalf("failed to create Terminal: %s", err.Error())
			}
			term.Start()
			var strout []string

			out := term.Out()
			errs := term.Errors()
			// count := 0

			// exec each cmd - sleep to give time for other proc to return
			go func() {
				for _, c := range tc.cmds {
					term.RunCommand([]byte(c))
					time.Sleep(time.Millisecond * time.Duration(250))
				}
				cncl()
			}()

			// count number of output reads - use ctx channel to signal we can move on
			done := ctx.Done()
		Outer:
			for {
				select {
				case b := <-out:
					strout = append(strout, string(b))
				case <-done:
					break Outer
				}
			}

			select {
			case err = <-errs:
				t.Errorf("unexpected error during run/cancel: %w", err)
			default:
				// empty case since this is expected for the test
			}

			if len(strout) != tc.expected {
				t.Errorf("expected %d reads, got %d", tc.expected, len(strout))
			}

		})
	}
}
