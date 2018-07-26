#!/bin/bash

# script to tell bazel what version to set on the client
VERSION=$(cd cmd/versioner && go run versioner.go)
echo VERSION ${VERSION}
if [ -z "$CI_DOCKER_TAG" ]; then
    # Non-CI build
    DOCKERTAG=latest
else
    DOCKERTAG=$CI_DOCKER_TAG
fi

REGISTRY=${CI_REGISTRY:-$(hostname).local:80}
REPOSITORY=${CI_REPOSITORY:-dotmesh}
echo DOCKERTAG ${DOCKERTAG}
echo CI_REGISTRY ${REGISTRY}
echo CI_REPOSITORY ${REPOSITORY}
echo CI_DOCKER_SERVER_IMAGE ${CI_DOCKER_SERVER_IMAGE}
echo CI_DOCKER_SERVER_IMAGE ${CI_DOCKER_SERVER_IMAGE} 1>&2
