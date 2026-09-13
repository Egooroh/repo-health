FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/repo-health ./cmd/repo-health

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --create-home app && mkdir /data && chown app:app /data
COPY --from=build /out/repo-health /usr/local/bin/repo-health
USER app
ENV HTTP_ADDR=0.0.0.0:8080 DATABASE_PATH=/data/repo-health.db
EXPOSE 8080
ENTRYPOINT ["repo-health"]
