IMAGE_NAME = isukim/ds-hw-1-api-server
TAG = latest

.PHONY: build

build:
	docker build -t $(IMAGE_NAME):$(TAG) .

.PHONY: clean

clean:
	docker rmi $(IMAGE_NAME):$(TAG)