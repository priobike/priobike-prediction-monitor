FROM bikenow.vkw.tu-dresden.de/priobike/priobike-nginx:v1.0

WORKDIR /app
COPY . .

# Install bash and jq (used in the healthcheck)
RUN apt-get update && apt-get install -y bash jq

# Install Go
RUN apt-get install curl -y
RUN curl -O https://dl.google.com/go/go1.19.1.linux-amd64.tar.gz
RUN tar xvf go1.19.1.linux-amd64.tar.gz
RUN chown -R root:root ./go
RUN mv go /usr/local

RUN /usr/local/go/bin/go mod download
RUN /usr/local/go/bin/go build -o main .

RUN ["chmod", "+x", "./healthcheck.sh"]
RUN ["chmod", "+x", "./main"]
HEALTHCHECK --interval=90s --timeout=85s --retries=1 --start-period=2s CMD ./healthcheck.sh

EXPOSE 80

# Run nginx in background. This makes sure that the container exits if we panic in the Go code.
CMD nginx && /app/main
