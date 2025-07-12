{
  description = "kiln - Secure environment variable management tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        version = "1.0.0";
        commit = self.rev or self.dirtyRev or "unknown";
        date = builtins.readFile (
          pkgs.runCommand "timestamp" { } ''
            date -u +"%Y-%m-%dT%H:%M:%SZ" > $out
          ''
        );

        ldflags = [
          "-X main.version=${version}"
          "-X main.commit=${commit}"
          "-X main.date=${date}"
        ];
      in
      {
        packages = {
          default = self.packages.${system}.kiln;

          kiln = pkgs.buildGoModule rec {
            inherit version ldflags;

            pname = "kiln";
            src = ./.;
            vendorHash = "sha256-r66QvgW5freuMPJEmUh5lumi9I9xKkKiDyd3aTe8aBQ=";

            env.CGO_ENABLED = if doCheck then "1" else "0";
            nativeBuildInputs = pkgs.lib.optionals doCheck [ pkgs.gcc ];

            doCheck = pkgs.stdenv.buildPlatform.canExecute pkgs.stdenv.hostPlatform;
            checkFlags = [
              "-v"
              "-race"
              "-bench=."
              "-benchmem"
            ];

            installPhase = ''
              runHook preInstall
              install -Dm755 $GOPATH/bin/kiln $out/bin/kiln
              runHook postInstall
            '';

            meta = with pkgs.lib; {
              description = "Secure environment variable management tool";
              homepage = "https://github.com/thunderbottom/kiln";
              license = licenses.mit;
              maintainers = [ ];
              mainProgram = "kiln";
            };
          };

          kiln-docs = pkgs.stdenv.mkDerivation {
            inherit version;

            pname = "kiln-docs";
            src = ./docs;

            nativeBuildInputs = with pkgs; [
              nodejs
              nodePackages.npm
            ];

            buildPhase = ''
              runHook preBuild
              npm ci --prefer-offline
              npm run build
              runHook postBuild
            '';

            installPhase = ''
              runHook preInstall
              mkdir -p $out/share/doc/kiln
              cp -r dist/* $out/share/doc/kiln/
              runHook postInstall
            '';

            meta = with pkgs.lib; {
              description = "Documentation for kiln secure environment variable management tool";
              homepage = "https://kiln.sh";
              license = licenses.mit;
            };
          };
        };

        devShells = {
          default = pkgs.mkShell {
            name = "kiln-dev-shell";

            nativeBuildInputs = with pkgs; [
              go_1_23
              go-tools
              gopls
              delve
              gnumake
              golangci-lint
            ];

            env = {
              CGO_ENABLED = "0";
              GOPATH = "${toString ./.}/.go";
              GOCACHE = "${toString ./.}/.cache/go-build";
            };

            shellHook = ''
              echo "kiln development environment"
              echo "Go version: $(go version)"
              echo "GOPATH: $GOPATH"
              echo ""
              echo "Available commands:"
              echo "  make build     - Build kiln binary"
              echo "  make test      - Run tests with coverage"
              echo "  make lint      - Run golangci-lint"
              echo "  make dev ARGS='--help'  - Run kiln in development mode"
              echo "  go run . --help         - Alternative way to run kiln"
              echo ""
              echo "Build targets:"
              echo "  make build-all    - Build for all platforms"
              echo "  make build-linux  - Build for Linux (amd64/arm64)"
              echo "  make build-darwin - Build for macOS (amd64/arm64)"
              echo ""
            '';
          };

          docs = pkgs.mkShell {
            name = "kiln-docs-dev-shell";

            nativeBuildInputs = with pkgs; [
              nodejs
              nodePackages.npm
            ];

            shellHook = ''
              echo "kiln documentation development environment"
              echo "Node.js version: $(node --version)"
              echo "npm version: $(npm --version)"
              echo ""
              echo "Available commands:"
              echo "  cd docs && npm install  - Install dependencies"
              echo "  cd docs && npm run dev  - Start Astro development server"
              echo "  cd docs && npm run build - Build documentation"
              echo "  cd docs && npm run preview - Preview built docs"
              echo ""
              echo "Documentation will be available at http://localhost:4321"
              echo ""
            '';
          };
        };

        formatter = pkgs.nixpkgs-fmt;

        checks = {
          kiln-build = self.packages.${system}.kiln;

          go-fmt =
            pkgs.runCommand "go-fmt-check"
              {
                nativeBuildInputs = [ pkgs.go_1_23 ];
              }
              ''
                cd ${./.}
                if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then
                  echo "Go files are not formatted. Run 'go fmt ./...'"
                  exit 1
                fi
                touch $out
              '';

          go-test =
            pkgs.runCommand "go-test"
              {
                nativeBuildInputs = [
                  pkgs.go_1_23
                  pkgs.gcc
                ];
                env.CGO_ENABLED = "1";
              }
              ''
                cd ${./.}
                go test -v -race -bench=. -benchmem -coverprofile=coverage.out ./...
                touch $out
              '';

          go-lint =
            pkgs.runCommand "go-lint-check"
              {
                nativeBuildInputs = with pkgs; [
                  go_1_23
                  golangci-lint
                ];
                env.CGO_ENABLED = "0";
              }
              ''
                cd ${./.}
                golangci-lint run ./...
                touch $out
              '';

          docs-build = self.packages.${system}.docs;
        };

        apps = {
          default = self.apps.${system}.kiln;

          kiln = flake-utils.lib.mkApp {
            drv = self.packages.${system}.kiln;
          };

          dev = flake-utils.lib.mkApp {
            drv = pkgs.writeShellScriptBin "kiln-dev" ''
              cd ${toString ./.}
              exec ${pkgs.go_1_23}/bin/go run . "$@"
            '';
          };
        };
      }
    );
}
