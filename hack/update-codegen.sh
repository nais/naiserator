#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

CODEGEN_PKG=${GOPATH}/src/k8s.io/code-generator
NAISERATOR_GOPATH=${GOPATH}/src/github.com/nais/naiserator

# Since code-generator doesn't support go modules yet, we work around this
# by copying the repo files into GOPATH, generating files, and moving it back.
# https://github.com/kubernetes/kubernetes/issues/67566

go get k8s.io/code-generator

cp -r ./* ${NAISERATOR_GOPATH}/

${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/nais/naiserator/pkg/client github.com/nais/naiserator/pkg/apis \
  naiserator:v1alpha1

cp -r ${NAISERATOR_GOPATH}/* ./
