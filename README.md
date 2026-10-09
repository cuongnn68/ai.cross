# ai-cross

- Sync AI coding rules (e.g., “use TypeScript”) into agent instruction files such as `AGENTS.md` and `CLAUDE.md`.
- Apply rules to one project or your user account with one command.
- Back up existing rules and restore previous versions.

## Users

### Install

- Install on Windows from a regular PowerShell terminal; no admin rights or Go installation required:

  ```powershell
  $installer = Join-Path $env:TEMP 'ai-cross-install.ps1'
  Invoke-WebRequest -UseBasicParsing https://raw.githubusercontent.com/cuongnn68/ai.cross/main/scripts/install.ps1 -OutFile $installer
  & $installer
  ai-cross help
  ```

- If script execution is blocked, allow it for this PowerShell session and run `& $installer` again:

  ```powershell
  Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
  ```

- The installer downloads the latest Windows release, verifies SHA-256, and installs to `%LOCALAPPDATA%\ai-cross\bin`.
  - Adds the install dir to user `PATH` and the current PowerShell session; restart other terminals afterward.
  - Supports x64 and ARM64; rerun the installer to update.
  - Downloads require public access to the repo and a successful Windows release build.
  - Organization-enforced execution policies may still block scripts.

- Install with Homebrew on macOS / Linux:

  ```sh
  brew tap cuongnn68/ai.cross https://github.com/cuongnn68/ai.cross.git
  brew install --HEAD cuongnn68/ai.cross/ai-cross
  ai-cross help
  ```

- The Homebrew formula builds `main` and installs Go as a build dependency; use `--HEAD`.
- The formula must be available on GitHub, and you need access to the repo.
- Upgrade:

  ```sh
  brew update && brew upgrade --fetch-HEAD cuongnn68/ai.cross/ai-cross
  ```

- Uninstall with `brew uninstall ai-cross`; instruction files and `~/.ai.cross` remain.
- For a local build, see [developer setup](#setup).

### Apply instructions

- Check agent locations, then apply an instruction file:

  ```sh
  ai-cross status --local
  ai-cross apply --local --file instructions.md
  ```

- Other input sources and project selection:

  ```sh
  ai-cross apply --global --clipboard
  ai-cross apply --local --url https://example.com/instructions.md
  ai-cross apply --local --url https://example.com/instructions.md --save-url
  ai-cross apply --local --saved-url
  ai-cross apply --local --project /path/to/repo
  printf 'Use concise answers.\n' | ai-cross apply --local
  ```

- `apply` and `restore` require `--local` or `--global`.
  - Local scope uses cwd or `--project`; it does not search parent repos.
  - Global scope uses the home dir.
  - `status` and `histories` default to local scope.
- With no source option, interactive input opens a multiline editor; Ctrl+S submits, Esc cancels.
- Clipboard requires `pbpaste` on macOS, PowerShell on Windows, or `wl-paste`, `xclip`, or `xsel` on Linux.
- `--url` downloads instructions from an HTTP(S) URL; use a raw text or Markdown link.
  - The response body is applied as-is; redirects are followed, with a 30-second timeout.
  - Failed downloads, non-2xx responses, and empty instructions are rejected before applying.
  - Add `--save-url` to save the URL in config when the apply succeeds.
  - Use `--saved-url` to download fresh instructions from the saved URL and apply them again.
  - Global and local URLs are separate; local scope has one shared URL across projects, and `--project` selects where to apply it.
  - `--saved-url` fails if the selected scope has no saved URL. Normal applies keep the saved URL unless `--save-url` replaces it.
  - Choose only one source: `--file`, `--url`, `--saved-url`, `--clipboard`, or `--input`.
- Apply defaults to agents detected on PATH, using the same detection as `status`.
  - `--all` applies to every registered agent regardless of detection.
  - `--agents codex,claude` applies only to the specified agents regardless of detection.
  - Agent selection accepts case-insensitive names from `status -v` or registered CLI commands, eg `--agents "Roo Code,cursor-agent"`.
  - `--all` and `--agents` cannot be combined; unknown agent names are rejected.
  - Config `additional` and `ignore` still apply in every selection mode.
  - Shared instruction paths affect every agent that reads them, eg `AGENTS.md`.
  - If no target paths remain, Apply exits before reading input or writing files.
  - `status` checks executables on PATH; IDE extensions and cloud installations may not be detected.
  - See the [agent location reference](doc/ai-agent-instruction-files.md) for supported paths.

### Remove instructions

- Remove every registered agent's instruction files in the selected scope:

  ```sh
  ai-cross black-hole --local
  ai-cross black-hole --local --project /path/to/repo
  ai-cross black-hole --global
  ```

- Requires `--local` or `--global`; includes agents not detected on PATH.
- Includes config `additional` paths and removes files even if listed in `ignore`.
- Uses the registered paths for the selected scope; does not search nested projects or change rules stored in agent settings.
- Backs up existing files before deletion; use the printed backup name with `restore` to recover.
- Preserves unrelated files and applies the existing scope, symlink, locking, and rollback checks.

### Config

- Use `~/.ai.cross/config.yml` or `config.yaml`; `config.yml` takes precedence.
- `ai-cross config` prints the config location.
- Add or ignore instruction paths with this config:

  ```yaml
  histories:
    global:
      url: https://example.com/global-instructions.md
      additional:
        - "~/VIBE.md"
      ignore:
        - "~/.claude/CLAUDE.md"
    local:
      url: https://example.com/project-instructions.md
      additional:
        - "./.team/VIBE.md"
      ignore:
        - "./CLAUDE.md"
  ```

- `additional` and `ignore` support file globs.
  - Existing matching files are overwritten.
  - Unmatched file globs create `ai-cross.md`, `.mdc`, or `.instructions.md` in a fixed parent dir.
  - Wildcard parent dirs must already have matching files; no fallback file is created there.
- `url` is optional and used only with `apply --saved-url`.
  - `--save-url` updates the existing config file, preserving other settings and comments; creates `config.yml` if neither filename exists.
  - The URL update shares the instruction write lock and transaction; failed applies keep the previous saved URL.
- Agents without fixed instruction paths need `additional` entries; arbitrary agent configs are not rewritten.
- Local paths must stay inside the project; global paths must stay inside the home dir.
  - Symlink instruction paths are rejected.
  - `CODEX_HOME` is respected when it stays inside the home dir.

### History and recovery

- List records, then restore a name from the output:

  ```sh
  ai-cross histories --local
  ai-cross histories --local --backups
  ai-cross histories --local --applies
  ```

- Restore a pre-apply backup using its timestamp, or an applied state using its `.md` name:

  ```sh
  ai-cross restore --local --name 20261003113058
  ai-cross restore --local --name 20261003113058.md
  ```

- Replace the example timestamp with your record name; use the same scope and project as the original apply.
- `restore list` is an alias for `histories`.
- History locations:
  - Global: `~/.ai.cross/instructions/global/`.
  - Local: `~/.ai.cross/instructions/project_location/FULL_PROJECT_PATH/`.
  - Windows local history includes the drive name to avoid collisions.
- Each record preserves recovery data:
  - `TIMESTAMP/`: backup files with relative dirs and a manifest of prior file states.
  - `TIMESTAMP.md`: supplied instructions.
  - `TIMESTAMP.json`: resulting file contents, paths, permissions, and absence state.
- Restore uses recorded paths even after config or registry changes, and backs up current state first.
  - Backup manifests track missing files, so restore can remove files created by apply.
- `--no-backup` skips the pre-apply backup; successful apply history is still recorded.
- Writes preserve existing permissions; new files use `0600`.
- Failed writes roll back and retain their backup when one was created.
  - A crash or power loss can interrupt a multi-file update.
  - A shared `~/.ai.cross/write.lock` prevents simultaneous writes.
  - After a crash, confirm no ai-cross process is active before removing the lock and restoring a backup.

## Developers

### Setup

- Install Go 1.26.5 or newer.
- From the repo root, build and inspect CLI help:

  ```sh
  go build -o bin/ai-cross .
  ./bin/ai-cross help
  ./bin/ai-cross status --local
  ```

- For Windows, build with `go build -o bin/ai-cross.exe .`.
- Local builds report version `dev`; release builds inject `main.version` via linker flags.

- Every push to `main` runs [.github/workflows/windows.yml](.github/workflows/windows.yml).
  - Builds x64 and ARM64 executables, checks the x64 CLI version, and publishes a GitHub Release with SHA-256 files.
  - Release tags and CLI versions use `main-COMMIT_SHA`; the newest release becomes the installer download source.
  - Requires GitHub Actions to be enabled and its token to have `contents: write`; no custom secret is needed.

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
- Reuse existing storage helpers for path validation and writes.
  - `resolve` and `safePath`: scope and symlink checks.
  - `targets`: registry/config expansion and deduplication.
  - `capture`, `writeFile`, and `transaction`: snapshots, temporary-file writes, and rollback.
  - `apply`: locking, backups, and applied-history records.
- Deployment references: [Homebrew](doc/deployment/brew.md) and [WinGet](doc/deployment/winget.md).

### Verification

- Run the Go tests yourself from the repo root:

  ```sh
  go test ./...
  ```

- After tapping and installing, run the Homebrew smoke test yourself:

  ```sh
  brew test cuongnn68/ai.cross/ai-cross
  ```
