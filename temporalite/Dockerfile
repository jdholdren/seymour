FROM alpine

# Download and unzip temporal
RUN apk add curl unzip
RUN curl -sSL https://github.com/temporalio/cli/releases/download/v1.3.0/temporal_cli_1.3.0_linux_arm64.tar.gz | tar -xz
RUN ls
RUN mv temporal /usr/local/bin && \
    rm -rf temporal_cli_1.3.0_linux_arm64

EXPOSE 8233
EXPOSE 7233

# Run sqlite temporal server
CMD ["temporal", "server", "start-dev", "--db-filename", "/data/temporal.db", "--ip", "0.0.0.0"]
