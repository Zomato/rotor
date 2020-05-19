FROM golang:1.14.2 AS golang

# Get go modules
COPY go.mod /go/src/github.com/turbinelabs/rotor/go.mod
COPY go.sum /go/src/github.com/turbinelabs/rotor/go.sum
RUN cd /go/src/github.com/turbinelabs/rotor &&  go mod download && go mod vendor

# Add src
COPY . /go/src/github.com/turbinelabs/rotor

# Install binaries
RUN go install github.com/turbinelabs/rotor/...

FROM phusion/baseimage:0.11

RUN apt-get update
RUN apt-get install gettext-base -y

# Clean up APT when done.
RUN apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Add support files
COPY --from=golang /go/bin/rotor* /usr/local/bin/
ADD rotor.sh /usr/local/bin/rotor.sh
RUN chmod +x /usr/local/bin/rotor.sh

COPY start_rotor.sh /usr/local/bin/start_rotor.sh

# best guess
EXPOSE 50000

# Use baseimage-docker's init system.
CMD ["/usr/local/bin/start_rotor.sh"]
