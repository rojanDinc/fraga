# Contributing to Fraga

Thank you for your interest in contributing to Fraga! This document provides guidelines for contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Features](#suggesting-features)
  - [Pull Requests](#pull-requests)
- [Code Style](#code-style)
- [Common Tasks](#common-tasks)
- [Pre-commit Checklist](#pre-commit-checklist)
- [License](#license)

## Getting Started

1. Fork the repository on GitHub.
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/fraga.git
   cd fraga
   ```
3. Add the upstream repository as a remote:
   ```bash
   git remote add upstream https://github.com/rojanDinc/fraga.git
   ```

## Development Setup

### Requirements

- Go 1.25.4 or later

### Building

```bash
# Build the CLI binary
go build -o fraga ./cmd/fraga

# Build and run
go run ./cmd/fraga
```

### Running Tests

```bash
# Run tests
make test

# Run tests + linting
make ci
```

### Linting and Formatting

```bash
make lint
```

## How to Contribute

### Reporting Bugs

If you find a bug, please open an issue using the [Bug Report template](https://github.com/rojanDinc/fraga/issues/new?template=bug_report.md).

Before reporting, please:
- Check if the bug has already been reported.
- Include steps to reproduce the issue.
- Mention your Go version and operating system.

### Suggesting Features

If you have an idea for a new feature, please open an issue using the [Feature Request template](https://github.com/rojanDinc/fraga/issues/new?template=feature_request.md).

Before suggesting, please:
- Check if the feature has already been suggested.
- Describe the use case and why it would be valuable.

### Pull Requests

1. Create a new branch from `main` for your changes:
   ```bash
   git checkout -b feature/my-feature
   ```
2. Make your changes, following the [Code Style](#code-style) guidelines.
3. Ensure all tests pass and the code is formatted.
4. Commit your changes with clear, descriptive messages.
5. Push to your fork:
   ```bash
   git push origin feature/my-feature
   ```
6. Open a Pull Request against the `main` branch of the upstream repository.

## Code Style

We follow standard Go conventions. For detailed guidelines, see [`Effective Go`](https://go.dev/doc/effective_go).

## Pre-commit Checklist

Before submitting a pull request, please ensure:

- [ ] Linting and tests pass `make ci`
- [ ] Module tidy: `go mod tidy`
- [ ] Binary builds: `go build -o fraga ./cmd/fraga`
- [ ] Comments added for exported items

## License

By contributing to Fraga, you agree that your contributions will be licensed under the project's license. See [LICENSE.md](./LICENSE.md) for details.
