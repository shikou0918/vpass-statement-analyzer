FROM golang:1.22-bookworm

WORKDIR /app

RUN useradd --create-home --shell /usr/sbin/nologin appuser \
  && mkdir -p /data \
  && chown -R appuser:appuser /app /data

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN chown -R appuser:appuser /app

EXPOSE 8080

USER appuser

CMD ["go", "run", "./cmd/server"]
