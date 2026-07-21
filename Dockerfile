# syntax=docker/dockerfile:1

FROM node:22-bookworm AS web
WORKDIR /web
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -tags embed -o /cred .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /cred /cred
EXPOSE 8080
ENTRYPOINT ["/cred", "web"]
