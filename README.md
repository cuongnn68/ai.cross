# ai-cross

Sync instructions across AI coding agents. Registry: [agent locations](doc/ai-agent-instruction-files.md). Requirements: [spec](doc/requirement.md).

Install from GitHub with Homebrew (macOS / Linux):

```sh
brew tap cuongnn68/ai.cross https://github.com/cuongnn68/ai.cross.git
brew install --HEAD cuongnn68/ai.cross/ai-cross
ai-cross help
```

- The [formula](Formula/ai-cross.rb) builds `main` from source and installs Go as a build dependency.
- Requires the formula to be pushed to GitHub first; access to the repo is required.
- No tagged release exists yet, so `--HEAD` is required.
- Upgrade: `brew update && brew upgrade --fetch-HEAD cuongnn68/ai.cross/ai-cross`.
- Uninstall: `brew uninstall ai-cross` (instruction files and `~/.ai.cross` remain).
- The explicit tap URL uses this repo directly ([Homebrew tap docs](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)).

Build locally and use:

```sh
go build -o bin/ai-cross .
./bin/ai-cross status --local
./bin/ai-cross status --local --verbose
./bin/ai-cross apply --local --file instructions.md
./bin/ai-cross apply --global --clipboard
./bin/ai-cross apply --local --project /path/to/repo
printf 'Use concise answers.\n' | ./bin/ai-cross apply --local
./bin/ai-cross histories --local
./bin/ai-cross restore --local --name 20261003113058
./bin/ai-cross restore --local --name 20261003113058.md
./bin/ai-cross help
```

- Requires Go 1.26.5 or newer; macOS, Linux, Windows.
- Bubble Tea provides multiline input: Ctrl+S submits; Esc cancels.
- `apply` and `restore` require explicit `--global` or `--local`.
- Local scope uses cwd or `--project`; it does not search parent repos.
- Applies to all registry paths, regardless of installation detection.
- `status` detects executables on PATH; IDE extensions/cloud installations cannot be reliably detected this way.
- `status` shows scope, the number of agents detected on PATH, and their names; `-v` / `--verbose` shows version, paths, and per-agent details.
- Glob rules overwrite existing matching files; empty rule dirs receive `ai-cross.md` (or the matching extension). Mode-specific wildcard dirs must already exist.
- Settings-only agents and user-selected instruction files need config `additional` paths. Arbitrary agent configs are not rewritten.
- Local paths stay within the project; global paths stay within the home dir. Symlink instruction paths are rejected.
- `CODEX_HOME` is respected when it is inside the home dir.
- Existing file permissions are preserved; new files use `0600`.

Config: `~/.ai.cross/config.yml` or `config.yaml` (`config.yml` takes precedence).

```yaml
histories:
  global:
    additional:
      - "~/VIBE.md"
    ignore:
      - "~/.claude/CLAUDE.md"
  local:
    additional:
      - "./.team/VIBE.md"
    ignore:
      - "./CLAUDE.md"
```

- `additional` and `ignore` support file glob patterns.
- Backups preserve relative dirs beneath `~/.ai.cross/instructions/global/TIMESTAMP` or `~/.ai.cross/instructions/project_location/FULL_PROJECT_PATH/TIMESTAMP`.
- Windows project history includes the drive name to avoid collisions.
- `TIMESTAMP.md` stores supplied instructions; `TIMESTAMP.json` stores the exact resulting file contents, paths, permissions, and absence state.
- Backup manifests record missing files so restore can remove files created by apply.
- Restore uses recorded paths, independent of subsequent registry/config changes, and backs up the current state first.
- `--no-backup` skips the pre-apply backup; successful apply history remains.
- `histories --applies` and `histories --backups` filter records. `restore list` is an alias.
- Writes use temporary files and rename, with rollback on reported failures. Failed writes retain their backup for recovery. A process crash/power loss can interrupt a multi-file update; the filesystem cannot atomically rename multiple independent files.
- A shared `~/.ai.cross/write.lock` prevents simultaneous ai-cross writes. After a crash, confirm no ai-cross process is active before removing this lock and restoring the backup.
- Clipboard: `pbpaste` on macOS; PowerShell on Windows; `wl-paste`, `xclip`, or `xsel` on Linux.

Verification (run yourself):

```sh
go test ./...
```

Homebrew formula verification (run yourself after tapping):

```sh
brew test cuongnn68/ai.cross/ai-cross
```
