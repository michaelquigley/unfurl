package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/michaelquigley/df/dl"
	"github.com/michaelquigley/unfurl"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	inPlace bool
	verbose bool
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
}

var rootCmd *cobra.Command

func init() {
	dl.Init(dl.DefaultOptions().SetTrimPrefix("github.com/michaelquigley/"))
	rootCmd = newRootCmd(&rootOptions{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
	})
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "unfurl [file]",
		Short:        "surgical markdown unwrap",
		Long:         "unfurl reads markdown and collapses soft line breaks inside paragraph content, leaving every other construct intact.",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if opts.verbose {
				dl.Init(dl.DefaultOptions().
					SetTrimPrefix("github.com/michaelquigley/").
					SetLevel(slog.LevelDebug))
			}
		},
		RunE: opts.runRoot,
	}
	cmd.SetIn(opts.stdin)
	cmd.SetOut(opts.stdout)
	cmd.SetErr(opts.stderr)
	cmd.PersistentFlags().BoolVarP(&opts.inPlace, "in-place", "i", false, "rewrite the file argument in place")
	cmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "emit progress and diagnostics")
	return cmd
}

func (opts *rootOptions) runRoot(cmd *cobra.Command, args []string) error {
	if opts.inPlace {
		if len(args) == 0 {
			return fmt.Errorf("--in-place requires a file argument")
		}
		return rewriteInPlace(args[0])
	}
	if len(args) == 0 {
		if err := unfurl.Unfurl(opts.stdin, opts.stdout); err != nil {
			return fmt.Errorf("unfurl stdin: %w", err)
		}
		return nil
	}

	in, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("open %s: %w", args[0], err)
	}
	defer in.Close()

	if err := unfurl.Unfurl(in, opts.stdout); err != nil {
		return fmt.Errorf("unfurl %s: %w", args[0], err)
	}
	return nil
}

func rewriteInPlace(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	out, err := unfurl.UnfurlBytes(src)
	if err != nil {
		return fmt.Errorf("unfurl %s: %w", path, err)
	}
	if err := atomicWriteFile(path, out, info.Mode()); err != nil {
		return fmt.Errorf("rewrite %s: %w", path, err)
	}
	return nil
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpName := tmp.Name()
	closed := false
	cleanup := true
	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		closed = true
		return fmt.Errorf("close temp file: %w", err)
	}
	closed = true

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	cleanup = false
	return nil
}
