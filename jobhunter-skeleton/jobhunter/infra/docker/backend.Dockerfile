# Multi-stage build -> tiny final image (single Go binary).
#
# The builder image must be >= the `go` directive in backend/go.mod, or the
# build fails with "go.mod requires go >= 1.25".
FROM golang:1.25 AS build
WORKDIR /src

# Copy the module files first so `docker build` can cache the dependency
# download layer, and a change to a .go file doesn't re-fetch every module.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO off + netgo: produces a fully static binary, which is what makes the
# distroless base (no libc) work.
# -trimpath and -ldflags="-s -w" drop build paths and the symbol table.
RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -tags netgo \
        -ldflags="-s -w" \
        -o /bin/api ./cmd/api

FROM gcr.io/distroless/base-debian12

# Run unprivileged. distroless ships a `nonroot` user (uid 65532); the binary
# needs no write access to anything, so there's no reason to run as root.
USER nonroot:nonroot

COPY --from=build /bin/api /bin/api
EXPOSE 8080
ENTRYPOINT ["/bin/api"]
