# build environment
FROM golang:latest AS builder

WORKDIR /app
ENV CGO_ENABLED=0 
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o coord ./cmd/Coordinate
RUN go build -o map ./cmd/Map 
RUN go build -o shuffle ./cmd/Shuffle
RUN go build -o reduce ./cmd/Reduce

# deployment environment
FROM alpine:latest AS coord
COPY --from=builder /app/coord .
CMD ["./coord"]

FROM alpine:latest AS map
COPY --from=builder /app/map .
CMD ["./map"]

FROM alpine:latest AS shuffle
COPY --from=builder /app/shuffle .
CMD ["./shuffle"]

FROM alpine:latest AS reduce
COPY --from=builder /app/reduce .
CMD ["./reduce"]
