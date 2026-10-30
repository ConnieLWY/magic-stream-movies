# ---- build stage ----
FROM golang:1.27-alpine AS build
WORKDIR /src

# Copy only the dependency manifests first so this layer stays cached until they change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server . \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# ---- runtime stage: just the binaries and CA certificates, running as a non-root user ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /out/seed /app/
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
