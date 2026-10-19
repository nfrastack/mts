# mts

modern replacement of `ts` (moreutils).

Timestamp every line of a log stream without Perl dependency.

mts reads standard input, prepends a timestamp to each line, and writes to standard output.

Extra features:
  - run a command while keeping stdout and stderr stamped separately
  - tee output to a file
  - propagate the wrapped command exit code

## Maintainer

nfrastack <code@nfrastack.com>

## Table of Contents

- [Maintainer](#maintainer)
- [Table of Contents](#table-of-contents)
- [Installing](#installing)
  - [From Source](#from-source)
  - [Nix](#nix)
  - [Precompiled Binaries](#precompiled-binaries)
- [Usage](#usage)
  - [Filter Mode](#filter-mode)
  - [Exec Mode](#exec-mode)
  - [Flags](#flags)
  - [Format](#format)
  - [Privileges](#privileges)
  - [Timezones](#timezones)
- [Compatibility and Enhancements](#compatibility-and-enhancements)
- [Support](#support)
  - [Usage](#usage-1)
  - [Bugfixes](#bugfixes)
  - [Feature Requests](#feature-requests)
  - [Updates](#updates)
- [License](#license)

## Installing

### From Source

Clone this repository and compile with [GoLang 1.26 or later](https://golang.org):

```bash
make build
sudo make install
```

`make install` copies the binary to `$(PREFIX)/bin` (`/usr/local` by default
`DESTDIR` honoured eg `make install DESTDIR=/tmp/stage PREFIX=/usr`)

Install man pages optionally:

```bash
sudo make install-man    # optional: $(PREFIX)/share/man/man1/mts.1
```

### Nix

```bash
nix run . -- [flags] -- cmd args...   # try installing
nix build                             # ./result/bin/mts
```

Install man pages with `nix build .#mts^man`.

### Precompiled Binaries

Precompiled binaries are available for download from the [GitHub Releases](https://github.com/nfrastack/mts/releases) page. Binaries are created only for tagged releases.

Visit the [Releases](https://github.com/nfrastack/mts/releases) page and download the binary for your architecture.

#### Supported Architectures

- `x86_64`
- `aarch64`

## Usage

### Filter Mode

Classic `ts` behavior. Timestamps arrive when mts reads each line, and each line is flushed immediately.

Examples:

```bash
ping nfrastack.com | mts
make 2>&1 | mts -i '[%H:%M:%S]'
./super_long_script.sh | mts -s '%.T'
mts --tz America/Vancouver < app.log
```

### Exec Mode

Run a command, stamp its standard output and standard error separately while keeping the streams separate. Signals are forwarded to the child, and `mts` exits with the child exit code for transparency in scripts and `set -o pipefail` pipelines.

Examples:

```bash
mts -- make test
mts --prefix '[web]' --output /var/log/web.log -- ./bin/web
TZ=America/Vancouver mts -- ./bin/web
```

### Flags

```
-i               elapsed since last line (default format "%H:%M:%S")
-s               elapsed since start (default format "%H:%M:%S")
-m               accepted for ts -m scripts - changes nothing
-r               rewrite existing timestamps as relative ("15m5s ago") or reformat
-u, --utc        use UTC instead of local time
--tz NAME,
--timezone NAME  use named timezone (America/Vancouver)
                 overrides -u. Falls back to $TZ, then legacy $TIMEZONE, then system local
--iso8601        timestamps as ISO-8601 / RFC3339Nano
--rfc3339        same as --iso8601
--prefix STR     extra label between timestamp and line, such as "[web]"
--output FILE    tee output to FILE (append unless --truncate)
--truncate       truncate --output FILE instead of appending
--output-mode MODE
                 Octal permissions (eg 640) for --output
--run-user USER[:GROUP]
                 run child as user (name or id)
                 Group defaults to primary group
--output-user USER[:GROUP]
                 Ownership (name or id) for --output
                 Shorthand for --output-user USER + --output-group GROUP.
                 Defaults to --run-user when set
--output-group GROUP
                 Group ownership (name or id) for --output
                 Overrides group in --output-user
-q, --quiet      Do not write to console with --output
--version        Version information
```



### Format

The optional `format` argument uses `strftime` conversions plus the moreutils extensions `%.S`, `%.s` and `%.T` (subsecond resolution).

Default: `%b %d %H:%M:%S`. Incremental default: `%H:%M:%S`.

### Privileges

`--run-user USER[:GROUP]` runs the wrapped command as that user (name or numeric id).
  - Group defaults to the primary group.
  - Switching users needs privileges, without `mts` exits 1.
  - Only the primary uid and gid are set.
  - Names resolve via `/etc/passwd`, numeric ids always work.

`--output-user USER[:GROUP]` changes `--output` file ownership (name or id).
  - `USER:GROUP` is shorthand for `--output-user USER` + `--output-group GROUP`.
  - Defaults to `--run-user` when set, so `sudo mts --run-user app --output log` is enough for the common case.
  -  Use `--output-group GROUP` to override the group or change group only.

`--output-mode MODE` sets `--output` file permissions (octal eg 640).
  - Default without the flag is `0644` masked by umask.

```bash
sudo mts --run-user app --output-mode 640 --output /var/log/app.log -- ./bin/app
sudo mts --run-user app --output /var/log/app.log --output-user root:adm -- ./bin/app
```

### Timezones

Absolute timestamps use the system local time by default (`/etc/localtime` when `$TZ` is unset). Override per invocation with `--tz`, or force UTC with `-u/--utc`.

Precedence: `--tz` > `$TZ` > `$TIMEZONE` > `-u` > system local.

> Named zones rely on system zoneinfo (`tzdata` package). Systems without tzdata still work with localtime/UTC - named zones fail with error.
> `--tz` and `-u` together are rejected (use `--tz UTC`).

## Compatibility and Enhancements

Compatibility and differences with `ts` from moreutils

| Feature                              | `ts` | `mts`                                |
| ------------------------------------ | ---- | ------------------------------------ |
| default / custom format              | yes  | yes                                  |
| `-i` / `-s`                          | yes  | yes                                  |
| `-m`                                 | yes  | accepted for compat, changes nothing |
| `-r` (relative times)                | yes  | yes                                  |
| `-u` / `--iso8601`                   | no   | yes                                  |
| `--tz` / `$TZ`                       | no   | yes                                  |
| `--user` / `--file-mode`             | no   | yes                                  |
| `--prefix` / `--output`              | no   | yes                                  |
| exec mode with exit-code passthrough | no   | yes                                  |

## Support

### Usage

- The [Discussions board](../../discussions) is a great place for working with the community.

### Bugfixes

- Please submit a [Bug Report](issues/new) if something isn't working as expected. I'll do my best to issue a fix in short order.

### Feature Requests

- Feel free to submit a feature request; however, there is no guarantee that it will be added or at what timeline.

### Updates

- Best effort to track upstream dependency changes, with more priority if the tool is actively used on our end.

## License

BSD-3-Clause. See [LICENSE](LICENSE) for more details.
