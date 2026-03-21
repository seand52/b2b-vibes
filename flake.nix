{
  description = "B2B Orders API - Go backend + React frontend dev environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };

        # Go 1.25+ (use latest available in nixpkgs)
        go = pkgs.go_1_24;  # Adjust when 1.25 is available in nixpkgs

      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go backend
            go
            gotools
            gopls
            golangci-lint
            go-migrate  # golang-migrate CLI
            delve       # Go debugger

            # Node.js / Frontend
            nodejs_22
            pnpm
            nodePackages.typescript
            nodePackages.typescript-language-server

            # Database
            postgresql_16  # psql client and tools

            # Testing / Playwright (uses system chromium on NixOS)
            chromium

            # Docker (for testcontainers)
            docker
            docker-compose

            # Utilities
            jq
            curl
            httpie
            watchexec   # File watcher for dev
          ];

          # Environment variables
          shellHook = ''
            # Go configuration
            export GOPATH="$HOME/go"
            export GOBIN="$GOPATH/bin"
            export PATH="$GOBIN:$PATH"
            export CGO_ENABLED=0

            # pnpm configuration (NixOS-friendly)
            export PNPM_HOME="$HOME/.local/share/pnpm"
            export PATH="$PNPM_HOME:$PATH"

            # Playwright configuration for NixOS
            # Use system chromium instead of Playwright's bundled browser
            export PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH="${pkgs.chromium}/bin/chromium"

            # Load project .env if it exists
            if [ -f .env ]; then
              set -a
              source .env
              set +a
            fi

            echo "🚀 B2B Orders API dev environment loaded"
            echo ""
            echo "Backend commands:"
            echo "  cd backend && make run     - Run API server"
            echo "  cd backend && make test    - Run Go tests"
            echo "  cd backend && make build   - Build binary"
            echo ""
            echo "Frontend commands:"
            echo "  pnpm install               - Install dependencies"
            echo "  pnpm dev                   - Run frontend dev server"
            echo "  pnpm test                  - Run Vitest tests"
            echo ""
            echo "Playwright:"
            echo "  Uses system chromium - no install needed"
            echo "  Run: make test-e2e"
            echo ""
          '';

          # Skip Playwright browser download - we use system chromium on NixOS
          PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD = "1";
        };
      }
    );
}
