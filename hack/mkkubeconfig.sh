#!/bin/bash

TEAMNAME="${1}"
CLUSTER_NAME="${2}"
LDAP_USERNAME="${3}"
LDAP_PASSWORD="${4}"

print_help() {
  cat <<EOF

This script will create a service user for a team, generate a valid kubeconfig with it's token and put it into Vault.

usage: ./create_serviceuser.sh team_name cluster_name

team_name         name of the team that will use the service user
cluster_name      name of the cluster that the user will be used

EOF
}

if [[ "${1}" == "-h" || "${1}" == "--help"  || "${#}" -ne "2" ]]; then print_help && exit 0; fi

for prog in kubectl jq base64 curl; do
    which $prog >/dev/null 2>&1
    if test $? -ne 0; then
        echo "Cannot find '$prog' in path, please install it to continue"
        exit 1
    fi
done

set -e

# create service account, and get token from secret
kubectl --namespace default create serviceaccount serviceuser-${TEAMNAME}
TOKEN=$(kubectl --namespace default get secret $(kubectl --namespace default get serviceaccount serviceuser-${TEAMNAME} -o=jsonpath='{.secrets[0].name}') -o=jsonpath='{.data.token}' | base64 --decode)

# generate a valid JSON kubeconfig with service account token and cluster info
TMP_KUBECONFIG=$(mktemp)
k="kubectl config --kubeconfig=${TMP_KUBECONFIG}"
${k} set-credentials service-user --token=${TOKEN}
${k} set-cluster ${CLUSTER_NAME} --server=https://apiserver.${CLUSTER_NAME}.nais.io --insecure-skip-tls-verify
${k} set-context ${CLUSTER_NAME} --user=service-user --cluster=${CLUSTER_NAME} --namespace default
${k} use-context ${CLUSTER_NAME}
JSON_KUBECONFIG=$(mktemp)
${k} view -o=json > ${JSON_KUBECONFIG}

read -p "Enter LDAP username: " LDAP_USERNAME
read -sp "Enter LDAP password: " LDAP_PASSWORD

# wrap kubeconfig and output unwrap token
VAULT_TOKEN=$(curl -s -XPOST -d "{\"password\": \"${LDAP_PASSWORD}\"}" https://vault.adeo.no/v1/auth/ldap/login/${LDAP_USERNAME} | jq .auth.client_token | tr -d '"')
UNWRAP_TOKEN=$(curl -s -XPOST -H "X-Vault-Token: ${VAULT_TOKEN}" -H "X-Vault-Wrap-TTL: 24h" --data @${JSON_KUBECONFIG} https://vault.adeo.no/v1/sys/wrapping/wrap | jq .wrap_info.token | tr -d '"')
echo "To get a working kubeconfig for your service account, go to https://vault.adeo.no/ui/vault/tools/unwrap and enter the following unwrap token (note: only usable once, and will be deleted after 24 hours):"
echo "${UNWRAP_TOKEN}"

rm -f ${TMP_KUBECONFIG} ${JSON_KUBECONFIG}
