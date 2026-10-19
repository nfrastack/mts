// SPDX-FileCopyrightText: © 2026 Nfrastack <code@nfrastack.com>
//
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nfrastack/mts/internal/rstamp"
	"github.com/nfrastack/mts/internal/strftime"
)

var (
	Version     = "dev"
	buildCommit = "unknown"
	BuildTime   = "unknown"
)

const (
	defaultFormat            = "%b %d %H:%M:%S"
	defaultIncrementalFormat = "%H:%M:%S"
)

type options struct {
	incremental   bool // -i: since last line
	sinceStart    bool // -s: since start
	monotonic     bool // -m: accepted, changes nothing
	relative      bool // -r: rewrite to relative
	utc           bool
	tz            string // --tz: location override
	iso8601       bool
	rfc3339       bool
	prefix        string
	output        string
	truncate      bool
	outputModeStr string // --output-mode: exact perms for --output
	runUser       string // --run-user: child identity USER[:GROUP]
	outputUser    string // --output-user: --output owner USER[:GROUP]
	outputGroup   string // --output-group: --output group override
	quiet         bool
	showVersion   bool
	format        string
	formatSet     bool // format given on command line
}

type identity struct {
	uid int
	gid int
}

type stamper struct {
	mu    sync.Mutex
	opts  options
	start time.Time
	prev  time.Time
	loc   *time.Location
	rw    rstamp.Rewriter
}

func newStamper(opts options, loc *time.Location) *stamper {
	now := time.Now()
	return &stamper{opts: opts, start: now, prev: now, loc: loc}
}

func (s *stamper) now() time.Time {
	return time.Now().In(s.loc)
}

func (s *stamper) stampLocked() string {
	now := s.now()
	var out string
	switch {
	case s.opts.iso8601, s.opts.rfc3339:
		out = now.Format(time.RFC3339Nano)
	case s.opts.incremental:
		out = strftime.FormatDuration(now.Sub(s.prev), s.opts.format)
	case s.opts.sinceStart:
		out = strftime.FormatDuration(now.Sub(s.start), s.opts.format)
	default:
		out = strftime.Format(now, s.opts.format)
	}
	s.prev = now
	return out
}

func (s *stamper) rewriteLocked(loc *time.Location, line []byte) []byte {
	now := s.now()
	switch {
	case s.opts.iso8601, s.opts.rfc3339:
		return s.rw.RewriteLine(loc, now, line, func(_, _, cur time.Time) string {
			return cur.Format(time.RFC3339Nano)
		})
	case s.opts.incremental:
		return s.rw.RewriteLine(loc, now, line, func(_, prev, cur time.Time) string {
			return strftime.FormatDuration(cur.Sub(prev), s.opts.format)
		})
	case s.opts.sinceStart:
		return s.rw.RewriteLine(loc, now, line, func(first, _, cur time.Time) string {
			return strftime.FormatDuration(cur.Sub(first), s.opts.format)
		})
	case s.opts.formatSet:
		return s.rw.RewriteLine(loc, now, line, func(_, _, cur time.Time) string {
			return strftime.Format(cur, s.opts.format)
		})
	default:
		return s.rw.RewriteLine(loc, now, line, func(_, _, cur time.Time) string {
			return rstamp.Ago(now.Sub(cur))
		})
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `mts - modern timestamp filter

Usage:
  Filter:  mts [flags] [format] < input
  Exec:    mts [flags] -- cmd args...

Flags:
  -i               elapsed since last line (default format "%%H:%%M:%%S")
  -s               elapsed since start (default format "%%H:%%M:%%S")
  -m               accepted for ts -m scripts; changes nothing
  -r               rewrite existing timestamps as relative ("15m5s ago") or reformat
  -u, --utc        use UTC instead of local time
  --tz NAME,
  --timezone NAME  use named timezone (eg America/Vancouver);
  --iso8601        timestamps as ISO-8601 / RFC3339Nano
  --rfc3339        same as --iso8601
  --prefix STR     extra label between timestamp and line, such as "[web]"
  --output FILE    tee output to FILE (append)
  --truncate       truncate --output FILE instead of appending
  --output-mode MODE
                   permissions for --output (octal eg 640)
  --run-user USER[:GROUP]
                   run child as user (name or id - needs privilege)
                   Group defaults to primary group
  --output-user USER[:GROUP]
                   change ownership for --output (name or id).
                   Shorthand for --output-user USER + --output-group GROUP.
                   Default to --run-user when set
  --output-group GROUP
                   chown --output group (names or id).
                   Overrides group in --output-user
                   alone changes group only
  -q, --quiet      with --output - do not write to stdout/stderr console
  --version        print version and exit

Format uses strftime conversions plus moreutils extensions %%.S %%.s %%.T.
Default: "%s". Incremental default: "%s".

Exec mode preserves stdout/stderr separation, forwards signals, and exits
with the child's exit code.
`, defaultFormat, defaultIncrementalFormat)
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	// keep "-- cmd..." out of flag parsing.
	var flagArgs, cmdArgs []string
	if idx := indexDashDash(argv); idx >= 0 {
		flagArgs = argv[:idx]
		cmdArgs = argv[idx+1:]
	} else {
		flagArgs = argv
	}

	fs := flag.NewFlagSet("mts", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var opts options
	var utcShort bool
	fs.BoolVar(&opts.incremental, "i", false, "")
	fs.BoolVar(&opts.sinceStart, "s", false, "")
	fs.BoolVar(&opts.monotonic, "m", false, "")
	fs.BoolVar(&opts.relative, "r", false, "")
	fs.BoolVar(&utcShort, "u", false, "")
	fs.BoolVar(&opts.utc, "utc", false, "")
	fs.StringVar(&opts.tz, "tz", "", "")
	fs.StringVar(&opts.tz, "timezone", "", "")
	fs.BoolVar(&opts.iso8601, "iso8601", false, "")
	fs.BoolVar(&opts.rfc3339, "rfc3339", false, "")
	fs.StringVar(&opts.prefix, "prefix", "", "")
	fs.StringVar(&opts.output, "output", "", "")
	fs.BoolVar(&opts.truncate, "truncate", false, "")
	fs.StringVar(&opts.outputModeStr, "output-mode", "", "")
	fs.StringVar(&opts.runUser, "run-user", "", "")
	fs.StringVar(&opts.outputUser, "output-user", "", "")
	fs.StringVar(&opts.outputGroup, "output-group", "", "")
	fs.BoolVar(&opts.quiet, "q", false, "")
	fs.BoolVar(&opts.quiet, "quiet", false, "")
	fs.BoolVar(&opts.showVersion, "version", false, "")
	fs.Usage = usage
	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}
	opts.utc = opts.utc || utcShort

	if opts.showVersion {
		fmt.Printf("mts %s (%s, %s)\n", Version, buildCommit, BuildTime)
		fmt.Println("Copyright © 2026 Nfrastack <https://nfrastack.com>")
		fmt.Println("License BSD-3-Clause. Homepage: https://github.com/nfrastack/mts")
		return 0
	}
	if opts.incremental && opts.sinceStart {
		fmt.Fprintln(os.Stderr, "mts: -i and -s are mutually exclusive")
		return 2
	}
	if cmdArgs != nil && len(cmdArgs) == 0 {
		fmt.Fprintln(os.Stderr, "mts: missing command after --")
		return 2
	}

	positional := fs.Args()
	switch len(positional) {
	case 0:
		if opts.incremental || opts.sinceStart {
			opts.format = defaultIncrementalFormat
		} else {
			opts.format = defaultFormat
		}
	case 1:
		opts.format = positional[0]
		opts.formatSet = true
	default:
		fmt.Fprintln(os.Stderr, "mts: at most one format argument is allowed")
		return 2
	}
	if opts.quiet && opts.output == "" {
		fmt.Fprintln(os.Stderr, "mts: -q/--quiet requires --output")
		return 2
	}
	if opts.utc && opts.tz != "" {
		fmt.Fprintln(os.Stderr, "mts: -u/--utc and --tz are mutually exclusive (use --tz UTC)")
		return 2
	}
	loc, err := resolveLocation(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mts: %v\n", err)
		return 2
	}
	fileMode, modeSet, err := parseFileMode(opts.outputModeStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mts: %v\n", err)
		return 2
	}
	isExec := cmdArgs != nil
	if modeSet && opts.output == "" {
		fmt.Fprintln(os.Stderr, "mts: --output-mode requires --output")
		return 2
	}
	if (opts.outputUser != "" || opts.outputGroup != "") && opts.output == "" {
		fmt.Fprintln(os.Stderr, "mts: --output-user/--output-group require --output")
		return 2
	}
	if opts.runUser != "" && !isExec {
		fmt.Fprintln(os.Stderr, "mts: --run-user requires exec mode (mts -- cmd...)")
		return 2
	}
	runID, err := resolveRunUser(opts.runUser)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mts: %v\n", err)
		return 2
	}
	outputID, err := resolveOutputUser(opts.outputUser, opts.outputGroup)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mts: %v\n", err)
		return 2
	}
	if isExec && opts.output != "" && outputID == nil {
		outputID = runID
	}

	if isExec {
		return runExec(opts, loc, fileMode, modeSet, runID, outputID, cmdArgs)
	}
	return runFilter(opts, loc, fileMode, modeSet, outputID, os.Stdin, os.Stdout)
}

func parseFileMode(s string) (os.FileMode, bool, error) {
	if s == "" {
		return 0, false, nil
	}
	n, err := strconv.ParseUint(s, 8, 32)
	if err != nil || n > 0o7777 {
		return 0, false, fmt.Errorf("bad --output-mode %q (octal up to 7777)", s)
	}
	return os.FileMode(n), true, nil
}

func resolveRunUser(spec string) (*identity, error) {
	if spec == "" {
		return nil, nil
	}
	u, g, hasGroup := splitUserGroup(spec)
	if u == "" {
		return nil, fmt.Errorf("bad --run-user %q (want USER[:GROUP])", spec)
	}
	return lookupIdentity(u, g, hasGroup, "--run-user", true)
}

func resolveOutputUser(userSpec, groupOverride string) (*identity, error) {
	if userSpec == "" && groupOverride == "" {
		return nil, nil
	}
	u, g, hasGroup := splitUserGroup(userSpec)
	if userSpec != "" && u == "" {
		return nil, fmt.Errorf("bad --output-user %q (want USER[:GROUP])", userSpec)
	}
	if hasGroup && groupOverride != "" {
		return nil, fmt.Errorf("--output-user group %q conflicts with --output-group %q", g, groupOverride)
	}
	effGroup := g
	effHasGroup := hasGroup
	if groupOverride != "" {
		effGroup = groupOverride
		effHasGroup = true
	}
	return lookupIdentity(u, effGroup, effHasGroup, "--output-user/--output-group", false)
}

func splitUserGroup(spec string) (user, group string, hasGroup bool) {
	u, g, found := strings.Cut(spec, ":")
	return u, g, found
}

func lookupIdentity(userPart, groupPart string, hasGroup bool, flag string, requireUser bool) (*identity, error) {
	var uid = -1
	var gid = -1
	if userPart != "" {
		if n, err := strconv.Atoi(userPart); err == nil {
			uid = n
			gid = n
			if groupPart == "" {
				if u, lerr := user.LookupId(userPart); lerr == nil {
					if g, cerr := strconv.Atoi(u.Gid); cerr == nil {
						gid = g
					}
				}
			}
		} else {
			u, err := user.Lookup(userPart)
			if err != nil {
				return nil, fmt.Errorf("unknown user %q (%s)", userPart, flag)
			}
			uid, _ = strconv.Atoi(u.Uid)
			gid, _ = strconv.Atoi(u.Gid)
		}
	} else if requireUser {
		return nil, fmt.Errorf("bad %s (want USER[:GROUP])", flag)
	}
	if hasGroup && groupPart != "" {
		if n, err := strconv.Atoi(groupPart); err == nil {
			gid = n
		} else {
			g, err := user.LookupGroup(groupPart)
			if err != nil {
				return nil, fmt.Errorf("unknown group %q (%s)", groupPart, flag)
			}
			gid, _ = strconv.Atoi(g.Gid)
		}
	}
	if uid == -1 && gid == -1 {
		return nil, fmt.Errorf("bad %s (want USER[:GROUP])", flag)
	}
	return &identity{uid: uid, gid: gid}, nil
}

func resolveLocation(opts options) (*time.Location, error) {
	if opts.tz != "" {
		loc, err := time.LoadLocation(opts.tz)
		if err != nil {
			return nil, fmt.Errorf("unknown timezone %q (is tzdata installed?)", opts.tz)
		}
		return loc, nil
	}
	if env := os.Getenv("TZ"); env != "" {
		loc, err := time.LoadLocation(env)
		if err != nil {
			return nil, fmt.Errorf("unknown timezone %q in $TZ (is tzdata installed?)", env)
		}
		return loc, nil
	}
	if env := os.Getenv("TIMEZONE"); env != "" {
		loc, err := time.LoadLocation(env)
		if err != nil {
			return nil, fmt.Errorf("unknown timezone %q (is tzdata installed?)", env)
		}
		return loc, nil
	}
	if opts.utc {
		return time.UTC, nil
	}
	return time.Local, nil
}

// stdin -> stdout (+ file).
func runFilter(opts options, loc *time.Location, fileMode os.FileMode, modeSet bool, outputID *identity, in io.Reader, out io.Writer) int {
	st := newStamper(opts, loc)

	var writers []io.Writer
	if !opts.quiet {
		writers = append(writers, out)
	}
	var f *os.File
	if opts.output != "" {
		var err error
		f, err = openOutput(opts.output, opts.truncate, fileMode, modeSet, outputID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mts: %v\n", err)
			return 1
		}
		defer f.Close()
		writers = append(writers, f)
	}
	dst := io.MultiWriter(writers...)

	prefix := ""
	if opts.prefix != "" {
		prefix = opts.prefix + " "
	}

	br := bufio.NewReaderSize(in, 64*1024)
	var buf []byte
	for {
		chunk, err := br.ReadBytes('\n')
		if len(chunk) > 0 {
			hasNL := chunk[len(chunk)-1] == '\n'
			line := chunk
			if hasNL {
				line = chunk[:len(chunk)-1]
			}
			st.mu.Lock()
			buf = buf[:0]
			if st.opts.relative {
				buf = append(buf, prefix...)
				buf = append(buf, st.rewriteLocked(loc, line)...)
			} else {
				ts := st.stampLocked()
				buf = append(buf, ts...)
				buf = append(buf, ' ')
				buf = append(buf, prefix...)
				buf = append(buf, line...)
			}
			st.mu.Unlock()
			buf = append(buf, '\n')
			if _, werr := dst.Write(buf); werr != nil {
				fmt.Fprintf(os.Stderr, "mts: write error: %v\n", werr)
				return 1
			}
		}
		if err != nil {
			if err == io.EOF {
				return 0
			}
			fmt.Fprintf(os.Stderr, "mts: read error: %v\n", err)
			return 1
		}
	}
}

func runExec(opts options, loc *time.Location, fileMode os.FileMode, modeSet bool, runID, outputID *identity, cmdArgs []string) int {
	st := newStamper(opts, loc)

	var f *os.File
	if opts.output != "" {
		var err error
		f, err = openOutput(opts.output, opts.truncate, fileMode, modeSet, outputID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mts: %v\n", err)
			return 1
		}
		defer f.Close()
	}
	var fmu sync.Mutex

	prefix := ""
	if opts.prefix != "" {
		prefix = opts.prefix + " "
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = os.Stdin
	attr := &syscall.SysProcAttr{}
	isolated := stdinPiped()
	attr.Setpgid = isolated
	if runID != nil {
		attr.Credential = &syscall.Credential{
			// primary ids only; supplementary groups inherited.
			Uid:         uint32(runID.uid),
			Gid:         uint32(runID.gid),
			NoSetGroups: true,
		}
	}
	cmd.SysProcAttr = attr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mts: %v\n", err)
		return 1
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mts: %v\n", err)
		return 1
	}
	if err := cmd.Start(); err != nil {
		if runID != nil && os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "mts: cannot switch to --run-user %q (need privilege)\n", opts.runUser)
			return 1
		}
		fmt.Fprintf(os.Stderr, "mts: %v\n", err)
		return 1
	}
	forwardSignals(cmd, isolated)

	var wg sync.WaitGroup
	wg.Add(2)
	var stdoutErr, stderrErr error
	go func() {
		defer wg.Done()
		stdoutErr = copyStamped(st, stdoutPipe, os.Stdout, f, &fmu, prefix, opts.quiet)
	}()
	go func() {
		defer wg.Done()
		stderrErr = copyStamped(st, stderrPipe, os.Stderr, f, &fmu, prefix, opts.quiet)
	}()
	wg.Wait()

	exit := 0
	if err := cmd.Wait(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				exit = 128 + int(ws.Signal())
			} else {
				exit = ee.ExitCode()
			}
		} else {
			fmt.Fprintf(os.Stderr, "mts: %v\n", err)
			return 1
		}
	}
	if stdoutErr != nil {
		fmt.Fprintf(os.Stderr, "mts: stdout: %v\n", stdoutErr)
	}
	if stderrErr != nil {
		fmt.Fprintf(os.Stderr, "mts: stderr: %v\n", stderrErr)
	}
	if exit == 0 && (stdoutErr != nil || stderrErr != nil) {
		return 1
	}
	return exit
}

func copyStamped(st *stamper, src io.Reader, console io.Writer, f *os.File, fmu *sync.Mutex, prefix string, quiet bool) error {
	br := bufio.NewReaderSize(src, 64*1024)
	var buf []byte
	var firstErr error
	fail := func(err error) {
		if firstErr == nil {
			firstErr = err
		}
	}
	consoleOK := true
	fileOK := f != nil
	for {
		chunk, err := br.ReadBytes('\n')
		if len(chunk) > 0 {
			hasNL := chunk[len(chunk)-1] == '\n'
			line := chunk
			if hasNL {
				line = chunk[:len(chunk)-1]
			}
			st.mu.Lock()
			buf = buf[:0]
			if st.opts.relative {
				buf = append(buf, prefix...)
				buf = append(buf, st.rewriteLocked(st.loc, line)...)
			} else {
				ts := st.stampLocked()
				buf = append(buf, ts...)
				buf = append(buf, ' ')
				buf = append(buf, prefix...)
				buf = append(buf, line...)
			}
			st.mu.Unlock()
			buf = append(buf, '\n')
			if !quiet && consoleOK {
				if _, werr := console.Write(buf); werr != nil {
					fail(werr)
					consoleOK = false
				}
			}
			if fileOK {
				fmu.Lock()
				_, werr := f.Write(buf)
				fmu.Unlock()
				if werr != nil {
					fail(werr)
					fileOK = false
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				return firstErr
			}
			fail(err)
			return firstErr
		}
	}
}

func openOutput(path string, truncate bool, fileMode os.FileMode, modeSet bool, id *identity) (*os.File, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if truncate {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}
	f, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		return nil, err
	}
	if id != nil {
		if err := f.Chown(id.uid, id.gid); err != nil {
			f.Close()
			return nil, err
		}
	}
	if modeSet {
		if err := f.Chmod(fileMode); err != nil {
			f.Close()
			return nil, err
		}
	}
	return f, nil
}

func forwardSignals(cmd *exec.Cmd, group bool) {
	ch := make(chan os.Signal, 8)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGWINCH)
	go func() {
		for sig := range ch {
			if cmd.Process == nil {
				continue
			}
			if group {
				if s, ok := sig.(syscall.Signal); ok {
					// negative pid targets the child's process group.
					_ = syscall.Kill(-cmd.Process.Pid, s)
					continue
				}
			}
			_ = cmd.Process.Signal(sig)
		}
	}()
}

func stdinPiped() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice == 0
}

func indexDashDash(argv []string) int {
	for i, a := range argv {
		if a == "--" {
			return i
		}
	}
	return -1
}
