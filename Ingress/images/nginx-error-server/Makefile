all: push

# TAG 0.0 shouldn't clobber any release builds
TAG = 0.2
PREFIX = aledbf/nginx-error-server

container:
	docker build -t $(PREFIX):$(TAG) .

push: container
	gcloud docker push $(PREFIX):$(TAG)
