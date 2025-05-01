FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
COPY *.go ./

RUN go build -o /logger

EXPOSE 8080

CMD ["/logger"]