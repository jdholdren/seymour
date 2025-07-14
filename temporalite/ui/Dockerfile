FROM alpine

RUN apk --no-cache add curl

# Download and unzip ui server
RUN curl -OL https://github.com/temporalio/ui-server/releases/download/v2.39.0/ui-server_2.39.0_linux_amd64.tar.gz
RUN tar -xzf ui-server_2.39.0_linux_amd64.tar.gz

# Install to bin?
RUN mv ui-server /usr/bin/temporal-ui-server

# Add config
COPY temporal-ui-server.yaml /etc/temporal/temporal-ui-server.yaml

# Run sqlite temporal server
CMD ["temporal-ui-server", "-r", "/", "-c", "/etc/temporal/", "-e", "temporal-ui-server", "start"]
