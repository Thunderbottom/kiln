<div align="center">
  <img src="./docs/src/assets/logo.svg" alt="Kiln Logo" width="200" height="100">
</div>

---

<div align="center">

[![kiln Documentation](https://img.shields.io/badge/kiln-documentation-blue)](https://kiln.sh)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Latest Release](https://img.shields.io/github/v/release/thunderbottom/kiln)](https://github.com/thunderbottom/kiln/releases)
[![Made at Zerodha Tech](https://zerodha.tech/static/images/github-badge.svg)](https://zerodha.tech)

</div>

---

kiln is a secure environment variable management tool that encrypts your sensitive configuration data using [age encryption](https://age-encryption.org/). It provides a simple, offline-first alternative to centralized secret management services, with role-based access control and support for both age and SSH keys, making it perfect for team collaboration and enterprise environments.

![kiln-demo.gid](./docs/public/kiln-demo.gif)

## Why?

Secret management is broken. Secrets get shared over chat, stored in plaintext files, or depend on external services that can fail during critical deployments. They remain vulnerable to anyone with file access, and deployments break when the secret management service is inaccessible.

Environment secrets should not depend on external services. They should be encrypted at rest, travel with code, and work offline. kiln solves this by encrypting environment variables into files that can be committed alongside the code. Each team member has their own key and can only decrypt authorized files. kiln can also execute commands by injecting the variables, so applications can access them directly.

No servers to maintain, no dependencies, no vendor lock-in. Secrets stay with code, encrypted and secure.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [Configuration](#configuration)
- [Team Collaboration](#team-collaboration)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

## Features

- **age & SSH Key Support**: Use existing SSH keys or generate new age keys with kiln
- **Multiple Environments**: Manage separate encrypted files for different environments
- **Team Collaboration**: Fine-grained role-based access control for team members and groups
- **Integrated Execution**: Run commands by injecting decrypted environment variables
- **Built-in Editor**: Edit environment files directly with your preferred editor
- **Cross-Platform**: Works on Linux, macOS, and Windows

## Installation

### Binary Releases

Download the latest binary for your platform from the [releases page](https://github.com/thunderbottom/kiln/releases), or install from source:

```shell
# Requires Go 1.23 or later:
go install github.com/thunderbottom/kiln@latest
```

Or with Nix:

```shell
nix run github:thunderbottom/kiln
```

## Quick Start

### 1. Generate a Key and Initialize the Configuration

```shell
# Generate a new age encryption key
$ kiln init key

# Create configuration with your public key
$ kiln init config --recipients "alice=$(cat ~/.kiln/kiln.key.pub)"

# Or use your existing SSH key
$ kiln init config --recipients "alice=$(cat ~/.ssh/id_ed25519.pub)"
```

### 2. Set and Get Variables

```shell
# Set an environment variable with a prompt
$ kiln set DATABASE_URL

# Or set it directly, if you are adventurous
$ kiln set API_KEY my-secret-key

# Get the value
$ kiln get API_KEY
```

### 3. Run Your Application

```shell
# kiln will inject the variables in your application's environment
$ kiln run -- your-application
```

## Usage

### Basic Commands

```shell
# Set variables
$ kiln set DATABASE_URL postgres://localhost/myapp

# Get the variable value
$ kiln get DATABASE_URL

# Edit the file directly in an editor
$ kiln edit --file production

# Export the variables to your shell
$ eval $(kiln export)

# Or to a file for your application to use
$ kiln export --format json > config.json

# Or apply variables directly to configuration templates
$ kiln apply --file production nginx.conf.template -o nginx.conf

# Or better, run the application with the secrets injected
$ kiln run -- npm start
$ kiln run --file production -- ./deploy.sh

# Share the variables with your team
$ kiln rekey --file staging --add-recipient "charlie=age1234..."
$ git add <env-file>

# Verify access
$ kiln info --verify
```

For all available command options, see `kiln [COMMAND] --help`.

## Configuration

kiln uses a `kiln.toml` configuration file with RBAC support:

```toml
# Named recipients with their public keys
[recipients]
alice = "age1234567890abcdef..."      # age key
bob = "ssh-ed25519 AAAAC3Nz..."       # SSH key
charlie = "age0987654321fedcba..."    # Another age key

# Groups for easier access management
[groups]
developers = ["alice", "bob"]
admins = ["alice"]
contractors = ["charlie"]

# Files with granular access control
[files]
default = { filename = ".kiln.env", access = ["*"] }
staging = { filename = "staging.env", access = ["developers"] }
production = { filename = "prod.env", access = ["admins"] }

# add access by groups or individual members
shared = { filename = "shared.env", access = ["alice", "contractors"] }
```

See the [Configuration Guide](https://kiln.sh/configuration/configuration-file)

### Environment Variables

- `KILN_PRIVATE_KEY_FILE`: Path to private key file
- `EDITOR`: Editor for `kiln edit` command

## Team Collaboration

### Adding Team Members

```shell
# Add a new team member with SSH key
$ kiln rekey --file staging --add-recipient "newdev=ssh-ed25519 AAAAC3Nz..."

# Add a new team member with age key
$ kiln rekey --file default --add-recipient "contractor=age1234567890abcdef..."
```

### Access Scenarios

```shell
# Developers can access staging
$ kiln get --key ~/.ssh/id_ed25519 --file staging DATABASE_URL

# Only admins can access production
$ kiln set --key ~/.kiln/admin.key --file production SECRET_KEY "prod-secret"

# Everyone defined in the recipients can access default environment
$ kiln export --file default --format shell
```

### Sharing Encrypted Files

```shell
# Commit encrypted files and configuration (safe to share)
$ git add .kiln.env staging.env prod.env kiln.toml
$ git commit -m "feat: add encrypted environment configuration"

# Team members with proper keys and access can decrypt their authorized files
$ kiln get DATABASE_URL --file staging  # Works if user is in developers group
```

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

### Development Setup

```shell
$ git clone https://github.com/thunderbottom/kiln.git
$ cd kiln
$ go mod download
$ make build
```

### Running Tests

```shell
$ make test
$ make test-coverage
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [Full documentation](https://kiln.sh)
- **FAQ**: [Frequently Asked Questions](https://kiln.sh/faq/)
- **Issues**: [GitHub Issues](https://github.com/thunderbottom/kiln/issues)
- **Discussions**: [GitHub Discussions](https://github.com/thunderbottom/kiln/discussions)

## Security

If you discover a security vulnerability, please submit it through GitHub's [Report a vulnerability](https://github.com/Thunderbottom/kiln/security) page. All security vulnerabilities will be promptly addressed.

---

_**Note**: kiln is designed for development and deployment workflows with enterprise-grade access control. Always follow your organization's security policies when handling sensitive data._

