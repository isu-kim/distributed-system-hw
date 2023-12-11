FROM golang:1.20
RUN apt-get update && apt-get install net-tools -y
RUN mkdir app/
WORKDIR app
COPY . .
RUN go get
RUN go build
RUN mkdir data
RUN mkdir config
RUN pwd
CMD ["./seph ./config/config.json"]
