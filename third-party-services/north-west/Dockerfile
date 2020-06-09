FROM golang:latest AS builder

# Download and install the latest release of dep
ADD https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

# Copy the code from the host and compile it
WORKDIR $GOPATH/src/terazo_parcel_service/third-party-services
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /app .

FROM alpine:latest
COPY --from=builder /app ./

EXPOSE 8080 8081 8082 8083 8084 8085

ENTRYPOINT ["./app"]
CMD ["north-west-service"]