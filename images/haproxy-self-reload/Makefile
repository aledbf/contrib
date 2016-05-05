all: push

# 0.0.0 shouldn't clobber any released builds
TAG = 0.1
PREFIX = gcr.io/google_containers/haproxy-self-reload

binary: main.go
	CGO_ENABLED=0 GOOS=linux godep go build -a -installsuffix cgo -ldflags '-w' -o haproxy-init

container: binary
	docker build -t $(PREFIX):$(TAG) .

push: container
	gcloud docker push $(PREFIX):$(TAG)

clean:
	# remove haproxy images
	docker rmi -f $(PREFIX):$(TAG) || true
