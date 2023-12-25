FROM golang:1.18

WORKDIR /go/src

COPY . .

RUN go build -o main main.go

CMD ["./main"]