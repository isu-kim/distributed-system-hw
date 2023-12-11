IMAGE_NAME = isukim/seph-client
TAG = latest

.PHONY: build

build:
	docker build -t $(IMAGE_NAME):$(TAG) .

.PHONY: clean

clean:
	docker rmi $(IMAGE_NAME):$(TAG)