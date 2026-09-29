FROM golang:1.25 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN GOOS=linux go build -o /cubpulse

FROM gcr.io/distroless/base-debian12 AS build-release-stage 

WORKDIR /

COPY --from=build-stage /cubpulse /cubpulse

USER nonroot:nonroot

ENTRYPOINT ["/cubpulse"]

