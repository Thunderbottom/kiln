# Contributing to kiln

Thank you for your interest in contributing to kiln! We welcome contributions of all kinds, from bug reports to feature implementations.

## Development

Requirements: Go 1.23+

1. Fork and clone the repository, and make a new branch: `$ git checkout https://github.com/thunderbottom/kiln -b [new-branch-name]`.
2. Add a feature, fix a bug, or refactor the code!
3. Write or update tests for the changes you made.
4. Run the tests and make sure all the tests pass.
5. Update the `README.md` if necessary.
6. Open a Pull RRequest with a comprehensive description of changes.

```bash
go mod download
make build
make test
```

## Guidelines

**Please follow these guidelines to get your work merged in**

- Follow standard Go conventions
- Add tests for new functionality, testing against the core feature
- Run `make lint` and fix linting issues before submitting a pull request
- Use clear commit messages, refer: https://www.conventionalcommits.org/en/v1.0.0/

## Security

Report security issues through the [Report a vulnerability](https://github.com/Thunderbottom/kiln/security) on GitHub. All security issues will be promptly addressed.

## License

Contributions are licensed under MIT.
