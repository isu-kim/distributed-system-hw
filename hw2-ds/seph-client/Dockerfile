FROM golang:1.20
RUN apt-get update && apt-get install net-tools -y
RUN mkdir app/
WORKDIR app
COPY . .
RUN go get
RUN go build
RUN mkdir config
CMD ["./seph-client ./config/config.json"]
