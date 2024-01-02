FROM golang:latest

WORKDIR /go/src

COPY . .

RUN go build -o main main.go

CMD ["./main"]