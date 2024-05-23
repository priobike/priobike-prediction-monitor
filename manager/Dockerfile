FROM golang:1.19.1-alpine

# Install bash and jq (used in the healthcheck)
RUN apk add --no-cache bash
RUN apk add --no-cache jq

WORKDIR /app

COPY . .

RUN ["chmod", "+x", "./healthcheck.sh"]
HEALTHCHECK --interval=90s --timeout=85s --retries=3 --start-period=2s CMD ./healthcheck.sh

RUN go mod download
RUN go build -o main .

CMD ["/app/main"]