FROM alpine

RUN apk --no-cache add curl

# Download and unzip temporal
RUN curl -OL https://github.com/temporalio/temporal/releases/download/v1.28.0/temporal_1.28.0_linux_amd64.tar.gz
RUN tar -xzf temporal_1.28.0_linux_amd64.tar.gz

# Install to bin?
RUN mv temporal-server /usr/bin/temporal-server

# Add config
COPY temporal-server.yaml /etc/temporal/temporal-server.yaml

# Run sqlite temporal server
CMD ["temporal-server", "-r", "/", "-c", "/etc/temporal/", "-e", "temporal-server", "start"]
