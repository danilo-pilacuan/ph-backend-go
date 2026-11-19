# syntax=docker/dockerfile:1
# Multi-stage para el backend Go de ProjectHub (fase 5 del plan de migracion).
# Build estatico y runtime distroless: imagen minima sin shell.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
ENV PORT=8100
EXPOSE 8100
USER nonroot:nonroot
ENTRYPOINT ["/server"]
