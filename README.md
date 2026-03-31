# profy

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Run any command with **profile-based environment variables** — loaded from external config, not from your repo.

```bash
profy <profile> <command>
```

## Why profy?

In real projects, you have multiple environments: `dev`, `sit`, `prod` — each with different database credentials, API keys, and secrets. Normally you'd manage `.env` files inside the repo (risky) or set them manually (tedious).

**profy** solves this by:

- Keeping all env configs **outside your repository** (e.g. `~/.profy/`) so secrets never get committed
- Letting you define **profiles** (`dev`, `sit`, `prod`) that each load a specific set of `.env` files
- **Merging** multiple env files in order — base config first, then profile-specific, then secrets
- **Validating** required keys before your app starts — no more runtime crashes from missing config
- **Watching** env files for changes and auto-restarting your app during development

Your app code stays clean. Just run `profy dev go run ./examples/app` and all environment variables are injected automatically.

## Install

```bash
go install github.com/dreamph/profy@latest
```

Or build from source:

```bash
git clone https://github.com/dreamph/profy.git
cd profy
go build -o profy .
```

## Quick start

### Step 1: Create `.profy.yml` in your project root

This tells profy which project config to look for:

```yaml
project_id: myapp
```

### Step 2: Set up external config

Create a config directory at `~/.profy/myapp/` with your env files:

```text
~/.profy/
└── myapp/
    ├── profy.json          # profile definitions
    ├── base.env           # shared across all profiles
    ├── dev.env            # dev-specific values
    ├── sit.env            # sit-specific values
    ├── prod.env           # prod-specific values
    └── secret/            # sensitive values (credentials, API keys)
        ├── dev.env
        ├── sit.env
        └── prod.env
```

Define your profiles in `profy.json`:

```json
{
  "configs": {
    "dev": {
      "files": ["base.env", "dev.env", "secret/dev.env"],
      "required_keys": ["APP_ENV", "APP_PORT", "DB_HOST", "DB_USER", "DB_PASSWORD"]
    },
    "sit": {
      "files": ["base.env", "sit.env", "secret/sit.env"],
      "required_keys": ["APP_ENV", "APP_PORT"]
    },
    "prod": {
      "files": ["base.env", "prod.env", "secret/prod.env"],
      "required_keys": ["APP_ENV", "APP_PORT"]
    }
  }
}
```

Each profile specifies:

- **`files`** — list of env files to load, merged in order (later files override earlier ones)
- **`required_keys`** — profy will fail fast if any of these keys are missing or empty

### Step 3: Run your app

```bash
profy dev go run ./examples/app
```

That's it. profy reads `.profy.yml` -> finds `myapp` config -> loads env files for `dev` profile -> injects them -> runs your command.

## Usage

```bash
profy [flags] <profile> <command> [args...]
```

| Flag | Default | Description |
| --- | --- | --- |
| `--override` | `false` | Override existing OS env vars with values from env files |
| `--watch-env` | `false` | Watch env files and auto-restart command when they change |
| `--watch-interval` | `1s` | How often to check for file changes (e.g. `500ms`, `2s`) |
| `--print-env` | `false` | Print all resolved env vars and exit (no command needed) |
| `--config-home` | `~/.profy` | Path to the external config directory |
| `--project-file` | `.profy.yml` | Path to the project config file |
| `-v, --verbose` | `false` | Show project, profile, and command info |

## Examples

```bash
# Basic: run API server with dev profile
profy dev go run ./examples/app

# Production: run compiled binary with prod env
profy prod ./myapp

# Hot-reload with air (rebuilds on Go file changes)
profy dev air -c .air.toml

# Auto-restart when env files change (profy watches the files itself)
profy --watch-env dev go run ./examples/app

# Debug: see what env vars profy would inject
profy --print-env dev

# Override: force env file values even if OS already has them set
profy --override prod ./myapp

# Custom config location (e.g. shared team config, CI/CD)
profy --config-home /etc/profy prod ./myapp
```

## How it works

```text
your-project/
└── .profy.yml                 # project_id: myapp
        │
        ▼
~/.profy/myapp/profy.json       # profile "dev" -> [base.env, dev.env, secret/dev.env]
        │
        ▼
   ┌────────────┬──────────────┬──────────────────┐
   │  base.env  │   dev.env    │  secret/dev.env   │
   │  APP_NAME  │  APP_ENV=dev │  DB_PASSWORD=xxx   │
   │  LOG_LEVEL │  APP_PORT    │  DB_USER=admin     │
   └────────────┴──────────────┴──────────────────┘
        │              │                │
        └──── merge (in order) ────────┘
                       │
                       ▼
              All env vars injected
                       │
                       ▼
         profy dev go run ./examples/app
```

- Files are merged **in order** — later files override earlier ones
- Variable expansion is supported: `DB_DSN=postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}`
- By default, existing OS env vars are **not** overridden (use `--override` to change this)
- `required_keys` are validated **before** your command starts — fail fast, not at runtime

## Env file format

Standard `.env` format with extras:

```bash
# Comments are supported
APP_NAME=myapp

# Quoted values (single or double)
GREETING="Hello World"
PATH_PREFIX='/api/v1'

# Variable expansion
DB_DSN=postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}

# Export prefix is allowed (stripped automatically)
export LOG_LEVEL=info

# Inline comments
API_TIMEOUT=30s  # request timeout
```

## License

MIT
