
<div align="center">
  <img src="./docs/logo.svg" alt="Kiln Logo" width="200" height="100">
</div>

---

<div align="center">

[![kiln Documentation](https://img.shields.io/badge/kiln-documentation-blue)](https://null.pointe.rs/kiln)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Latest Release](https://img.shields.io/github/v/release/thunderbottom/kiln)](https://github.com/thunderbottom/kiln/releases)

</div>

---

kiln is a secure environment variable management tool that encrypts your sensitive configuration data using [age encryption](https://age-encryption.org/). It provides a simple command-line interface for storing, retrieving, and deploying environment variables safely across all your infrastructure.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)

## Features

- **Age Encryption**: Uses modern, secure age encryption with X25519 keys
- **Multiple Environments**: Manage separate encrypted files for different environments
- **Team Collaboration**: Support for multiple recipients to share encrypted environment files
- **Simple Commands**: Intuitive CLI for setting, getting, and exporting variables
- **Integrated Execution**: Run commands by injecting decrypted environment variables
- **Built-in Editor**: Edit environment files directly with your preferred editor
- **Cross-Platform**: Works on Linux, macOS, and Windows

## Installation

### Binary Releases

Download the latest binary for your platform from the [releases page](https://github.com/thunderbottom/kiln/releases):

```shell
# Linux (x64)
curl -L https://github.com/thunderbottom/kiln/releases/latest/download/kiln-linux-amd64 -o kiln
chmod +x kiln
sudo mv kiln /usr/local/bin/

# macOS (x64)
curl -L https://github.com/thunderbottom/kiln/releases/latest/download/kiln-darwin-amd64 -o kiln
chmod +x kiln
sudo mv kiln /usr/local/bin/
```

### From Source

Requires Go 1.23 or later:

```shell
$ go install github.com/thunderbottom/kiln@latest
```

### Package Managers

```shell
# Nix (coming soon)
$ nix shell nixpkgs#kiln

# Arch Linux (AUR - coming soon)
$ yay -S kiln
```

## Quick Start

1. **Initialize a new project**:
   ```shell
   # Generate a new encryption key
   $ kiln init key

   # Or create an encrypted key, requiring a passphrase on every action
   $ kiln init key --encrypt

   # Create configuration with your public key
   $ kiln init config --public-keys ~/.kiln/kiln.key
   ```

2. **Set and get environment variables**:
   ```shell
   # Set an environment variable through a prompt
   $ kiln set DATABASE_URL

   # Or if you are feeling adventurous enough to skip the prompt
   $ kiln set API_KEY dragster-showgirl-overbite

   # And when you want to see what it's set to
   $ kiln get API_KEY
   ```

3. **Run your application**:
   ```shell
   # kiln will inject the variables in your application's environment
   $ kiln run -- your-application
   ```

## Usage

### Basic Commands

```shell
# Set a variable (prompts for value)
$ kiln set DATABASE_URL

# Set a variable with value
$ kiln set PORT 8080

# Get a variable
$ kiln get DATABASE_URL

# List all variables
$ kiln export --format json

# Edit variables in your editor
$ kiln edit

# Run command with environment
$ kiln run -- npm start
```

For all available command options, see `kiln [COMMAND] --help`.

### Working with Multiple Environments

```shell
# Edit the kiln.toml to add another environment
[files]
  staging = "staging.env"
  production = "production.env"

# Set variables for different environments
$ kiln set DATABASE_URL --file production
$ kiln set DATABASE_URL --file staging

# Run with specific environment
$ kiln run --file production -- ./deploy.sh
```

### Team Collaboration

```shell
# Add team member's public key
$ kiln rekey --add-recipient age1234567890abcdef... --file default

# Share the encrypted environment file
$ git add .kiln.env kiln.toml
$ git commit -m "chore: add encrypted environment configuration"
```

## Configuration

kiln uses a `kiln.toml` configuration file:

```toml
recipients = [
    "age1234567890abcdef...",  # Your public key
    "age0987654321fedcba..."   # Team member's public key
]

[files]
default = ".kiln.env"
production = ".kiln.prod.env"
staging = ".kiln.staging.env"
```

### Environment Variables

- `KILN_PRIVATE_KEY_FILE`: Path to private key file
- `EDITOR`: Editor for `kiln edit` command

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

- **Documentation**: [Full documentation](https://thunderbottom.github.io/kiln/)
- **Issues**: [GitHub Issues](https://github.com/thunderbottom/kiln/issues)
- **Discussions**: [GitHub Discussions](https://github.com/thunderbottom/kiln/discussions)

## Security

If you discover a security vulnerability, please submit it through GitHub's [Report a vulnerability](https://github.com/Thunderbottom/kiln/security) page. All security vulnerabilities will be promptly addressed.

---

_**Note**: kiln is designed for development and deployment workflows. Always follow your organization's security policies when handling sensitive data._
