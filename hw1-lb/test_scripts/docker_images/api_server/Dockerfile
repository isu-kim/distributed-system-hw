FROM golang:1.20
RUN apt-get update && apt-get install net-tools -y
RUN mkdir app/
WORKDIR app
COPY . .
RUN go mod init api-server
RUN go get
RUN go build
CMD ["./api-server"]
