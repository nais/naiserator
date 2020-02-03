#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

TEMP_DIR=`mktemp -d`
SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=$TEMP_DIR/cgen
export GOPATH=~/go
export GO111MODULE=on

git clone -b kubernetes-1.15.6 https://github.com/kubernetes/code-generator $CODEGEN_PKG

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/nais/naiserator/pkg/client github.com/nais/naiserator/pkg/apis \
  "nais.io:v1alpha1 networking.istio.io:v1alpha3 iam.cnrm.cloud.google.com:v1beta1 storage.cnrm.cloud.google.com:v1beta1 sql.cnrm.cloud.google.com:v1beta1" \
  --output-base ${TEMP_DIR} \
  --go-header-file ${SCRIPT_ROOT}/hack/boilerplate.go.txt

rsync -av ${TEMP_DIR}/github.com/nais/naiserator/ $SCRIPT_ROOT/

rm -rf $TEMP_DIR

# To use your own boilerplate text use:
#   --go-header-file ${SCRIPT_ROOT}/hack/custom-boilerplate.go.txt
