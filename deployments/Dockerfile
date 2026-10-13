FROM --platform=$BUILDPLATFORM docker.io/golang:1.27 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

WORKDIR /app/cmd/iot-agent

ARG TARGETARCH
ARG TARGETOS

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build

FROM registry.access.redhat.com/ubi9/ubi-minimal
WORKDIR /opt/diwise

# Unified config with 2 top sections: deviceprofiles + decoders.
COPY --chown=1001 assets/config/iot-agent.yaml /opt/diwise/config/iot-agent.yaml
COPY --chown=1001 assets/decoders/ /opt/diwise/config/decoders/
COPY --from=builder --chown=1001 /app/cmd/iot-agent/iot-agent /opt/diwise

RUN chown 1001 /opt/diwise

EXPOSE 8080
USER 1001

ENTRYPOINT ["/opt/diwise/iot-agent"]

