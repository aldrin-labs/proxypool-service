# core/signal_service

## Description

Notification service for user-defined events.

## Local deployment and building

Get sources for the service and dependencies

`go get -u -d -v gitlab.com/crypto_project/core/signal_service.git`

Latest versions is ok for now, might want to use dep and versioning later

To build the executables, you might want to use either

`go build`

in the project's root, which will build an executable right to the place where you invoked it, or

`go install`

which will build an executable to your $gopath/bin
