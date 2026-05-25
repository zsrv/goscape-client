# Build / test / lint container for goscape-client.
#
# SCOPE: headless development only — build, test, vet, lint. This image
# deliberately does NOT support RUNNING the client. The native window backend
# needs a display server (X11/Wayland), GPU/GL drivers, and an audio device,
# none of which are wired up here. Run the game on your host instead:
#
#     go run ./cmd/client 10 0 highmem members
#
# The build dependencies live in apt-packages.txt (single source of truth,
# also consumed by the Makefile `setup` target). They are required at COMPILE
# time, not just at runtime: pkg/jagex2/platform imports go-gl/glfw and
# go-gl/gl, so even *test* binaries link against these native libs.

# Matches the base previously pinned in devcontainer.json (official Go image,
# Debian trixie). The golang image is buildpack-deps based, so gcc/pkg-config/
# curl are already present; they are listed below anyway for explicitness.
FROM docker.io/golang:1.26.3-trixie

# CGO is mandatory for the native GLFW/go-gl backend on Linux (cgo bindings to
# the windowing/GL libs).
# The inherited server Makefile's blanket CGO_ENABLED=0 would break the build.
ENV CGO_ENABLED=1

# Install the shared dependency list. Strips comments/blank lines, then feeds
# the package names to apt. The COPY path works because the devcontainer build
# context defaults to the .devcontainer/ folder (and CI builds with that same
# context).
COPY apt-packages.txt /tmp/apt-packages.txt
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        $(grep -vE '^[[:space:]]*#|^[[:space:]]*$' /tmp/apt-packages.txt) \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/apt-packages.txt

# golangci-lint v2 (the v2 config schema is used in .golangci.yml). Must be a
# release with Go 1.26 support (>= the 2026-02 release; v2.12.2 is current as
# of 2026-05). Installed to /usr/local/bin so it is on PATH.
ARG GOLANGCI_LINT_VERSION=v2.12.2
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
    | sh -s -- -b /usr/local/bin "${GOLANGCI_LINT_VERSION}"
