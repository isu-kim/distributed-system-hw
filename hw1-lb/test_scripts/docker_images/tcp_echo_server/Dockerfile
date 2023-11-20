FROM golang:1.20
RUN apt-get update && apt-get install net-tools -y
RUN mkdir app/
WORKDIR app
COPY . .
CMD ["go run main.go"]
