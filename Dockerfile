FROM golang:alpine as builder
RUN apk update && apk add --no-cache git
COPY . $GOPATH/src/pollen/
WORKDIR $GOPATH/src/pollen/
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /go/bin/main .
FROM scratch
ENV POLLEN_REGION 90
ENV POLLEN_PARTREGION 92
COPY --from=builder /go/bin/main /app/
WORKDIR /app
CMD ["./main"]
