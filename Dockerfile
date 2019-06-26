############################
# STEP 1 build executable binary
############################
FROM golang@sha256:06ba1dae97f2bf560831497f8d459c68ab75cc67bf6fc95d9bd468ac259c9924 as builder
# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates
# Create appuser
RUN adduser -D -g '' appuser
RUN echo $(ls $GOPATH/src)
WORKDIR $GOPATH/src/gitlab.com/crypto_project/core/proxypool_service/
COPY . .
# Fetch dependencies.
#RUN GO111MODULE=on
#RUN git config --global url."https://gitlab+deploy-token-73239:2fWRgE1K7sVjzMjNZZm1@gitlab.com/".insteadOf "https://gitlab.com/"
#RUN go mod init gitlab.com/crypto_project/core/signal_service
#RUN apk add --no-cache openssh-client git
#RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan gitlab.com >> ~/.ssh/known_hosts
#RUN go get gitlab.com/crypto_project/core/signal_service.git
#RUN go get github.com/joho/godotenv
RUN GO111MODULE=on go mod download

#Build
RUN GO111MODULE=on GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o /go/bin/proxypool_service ./src/main.go

############################
# STEP 2 build a small image
############################
FROM scratch
# Import from builder.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
# Copy our static executable
COPY --from=builder /go/bin/proxypool_service /go/bin/proxypool_service
# Use an unprivileged user.
USER appuser
# Run the hello binary.
ENTRYPOINT ["/go/bin/proxypool_service"]
