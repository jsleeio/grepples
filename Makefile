IMAGEBASE=gcr.io/jsleeio-containers/grepples
TAG_TAG := $(if $(TAG),$(TAG),$(BRANCH))
FINAL_TAG := $(if $(TAG_TAG),$(TAG_TAG),latest)

.PHONY: docker

docker:
	docker build -t $(IMAGEBASE):$(FINAL_TAG) .
	docker push $(IMAGEBASE):$(FINAL_TAG)
