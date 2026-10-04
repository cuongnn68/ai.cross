# ai-cross

- Sync instruction files across AI coding agents, with backups and restore ([requirements](doc/requirement.md)).
- [Users](#users): install, apply instructions, configure paths, and restore history.
- [Developers](#developers): build locally, navigate the code, and verify changes.

## Users

### Install

- Install with Homebrew on macOS / Linux ([deployment guide](doc/deployment/brew.md)):

  ```sh
  brew tap cuongnn68/ai.cross https://github.com/cuongnn68/ai.cross.git
  brew install --HEAD cuongnn68/ai.cross/ai-cross
  ai-cross help
  ```

- The [formula](Formula/ai-cross.rb) builds `main` and installs Go as a build dependency; use `--HEAD`.
- The formula must be available on GitHub, and you need access to the repo ([deployment guide](doc/deployment/brew.md)).
- Upgrade ([deployment guide](doc/deployment/brew.md)):

  ```sh
  brew update && brew upgrade --fetch-HEAD cuongnn68/ai.cross/ai-cross
  ```

- Uninstall with `brew uninstall ai-cross`; instruction files and `~/.ai.cross` remain ([formula](Formula/ai-cross.rb), [storage](storage.go)).
- For a local build, see [developer setup](#setup).

### Apply instructions

- Check agent locations, then apply an instruction file ([CLI](main.go)):

  ```sh
  ai-cross status --local
  ai-cross apply --local --file instructions.md
  ```

- Other input sources and project selection ([CLI](main.go)):

  ```sh
  ai-cross apply --global --clipboard
  ai-cross apply --local --project /path/to/repo
  printf 'Use concise answers.\n' | ai-cross apply --local
  ```

- `apply` and `restore` require `--local` or `--global` ([CLI](main.go)).
  - Local scope uses cwd or `--project`; it does not search parent repos.
  - Global scope uses the home dir.
  - `status` and `histories` default to local scope.
- With no source option, interactive input opens a multiline editor; Ctrl+S submits, Esc cancels ([input](input.go)).
- Clipboard requires `pbpaste` on macOS, PowerShell on Windows, or `wl-paste`, `xclip`, or `xsel` on Linux ([clipboard handling](main.go)).
- Apply replaces contents at all target paths, regardless of detected installations ([CLI](main.go), [target resolution](storage.go)).
  - `status` checks executables on PATH; IDE extensions and cloud installations may not be detected ([registry](registry.go)).
  - See the [agent location reference](doc/ai-agent-instruction-files.md) for supported paths.

### Config

- Use `~/.ai.cross/config.yml` or `config.yaml`; `config.yml` takes precedence ([config loader](storage.go)).
- `ai-cross config` prints the config location ([CLI](main.go)).
- Add or ignore instruction paths with this config ([config schema](storage.go)):

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

- `additional` and `ignore` support file globs ([target resolution](storage.go)).
  - Existing matching files are overwritten.
  - Unmatched file globs create `ai-cross.md`, `.mdc`, or `.instructions.md` in a fixed parent dir.
  - Wildcard parent dirs must already have matching files; no fallback file is created there.
- Agents without fixed instruction paths need `additional` entries; arbitrary agent configs are not rewritten ([registry](registry.go), [target resolution](storage.go)).
- Local paths must stay inside the project; global paths must stay inside the home dir ([path validation](storage.go)).
  - Symlink instruction paths are rejected.
  - `CODEX_HOME` is respected when it stays inside the home dir ([registry](registry.go)).

### History and recovery

- List records, then restore a name from the output ([CLI](main.go)):

  ```sh
  ai-cross histories --local
  ai-cross histories --local --backups
  ai-cross histories --local --applies
  ```

- Restore a pre-apply backup using its timestamp, or an applied state using its `.md` name ([restore](main.go)):

  ```sh
  ai-cross restore --local --name 20261003113058
  ai-cross restore --local --name 20261003113058.md
  ```

- Replace the example timestamp with your record name; use the same scope and project as the original apply ([CLI](main.go)).
- `restore list` is an alias for `histories` ([CLI](main.go)).
- History locations ([storage](storage.go)):
  - Global: `~/.ai.cross/instructions/global/`.
  - Local: `~/.ai.cross/instructions/project_location/FULL_PROJECT_PATH/`.
  - Windows local history includes the drive name to avoid collisions.
- Each record preserves recovery data ([storage](storage.go)):
  - `TIMESTAMP/`: backup files with relative dirs and a manifest of prior file states.
  - `TIMESTAMP.md`: supplied instructions.
  - `TIMESTAMP.json`: resulting file contents, paths, permissions, and absence state.
- Restore uses recorded paths even after config or registry changes, and backs up current state first ([restore](main.go)).
  - Backup manifests track missing files, so restore can remove files created by apply.
- `--no-backup` skips the pre-apply backup; successful apply history is still recorded ([storage](storage.go)).
- Writes preserve existing permissions; new files use `0600` ([storage](storage.go)).
- Failed writes roll back and retain their backup when one was created ([storage](storage.go)).
  - A crash or power loss can interrupt a multi-file update.
  - A shared `~/.ai.cross/write.lock` prevents simultaneous writes.
  - After a crash, confirm no ai-cross process is active before removing the lock and restoring a backup.

## Developers

### Setup

- Install Go 1.26.5 or newer ([go.mod](go.mod)).
- From the repo root, build and inspect CLI help ([entry point](main.go)):

  ```sh
  go build -o bin/ai-cross .
  ./bin/ai-cross help
  ./bin/ai-cross status --local
  ```

- For Windows, build with `go build -o bin/ai-cross.exe .` ([entry point](main.go)).
- Local builds report version `dev`; release builds inject `main.version` via linker flags ([CLI](main.go), [formula](Formula/ai-cross.rb)).

### Code layout

- [main.go](main.go): CLI flags, command dispatch, history listing, restore, and clipboard input.
- [input.go](input.go): Bubble Tea multiline editor.
- [registry.go](registry.go): agent names, executables, and global/local instruction paths.
- [storage.go](storage.go): config, path validation, target resolution, snapshots, backups, locking, and writes.
- [storage_test.go](storage_test.go): backup/restore, history, rollback, path filtering, and write-lock tests.
- [Formula/ai-cross.rb](Formula/ai-cross.rb): Homebrew build, install, and smoke test.

### Working on changes

- Use the [requirements](doc/requirement.md) for expected behavior.
- For agent path changes, update the [registry](registry.go) and [agent location reference](doc/ai-agent-instruction-files.md).
- Reuse existing storage helpers for path validation and writes ([storage.go](storage.go)).
  - `resolve` and `safePath`: scope and symlink checks.
  - `targets`: registry/config expansion and deduplication.
  - `capture`, `writeFile`, and `transaction`: snapshots, temporary-file writes, and rollback.
  - `apply`: locking, backups, and applied-history records.
- Deployment references: [Homebrew](doc/deployment/brew.md) and [WinGet](doc/deployment/winget.md).

### Verification

- Run the Go tests yourself from the repo root ([tests](storage_test.go)):

  ```sh
  go test ./...
  ```

- After tapping and installing, run the Homebrew smoke test yourself ([formula test](Formula/ai-cross.rb)):

  ```sh
  brew test cuongnn68/ai.cross/ai-cross
  ```
