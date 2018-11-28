#!/bin/bash

TEAMNAME="${1}"
CLUSTER_NAME="${2}"
LDAP_USERNAME="${3}"
LDAP_PASSWORD="${4}"

print_help() {
  cat <<EOF

This script will create a service user for a team, generate a valid kubeconfig with it's token and put it into Vault.

usage: ./create_serviceuser.sh [team_name] [cluster_name] [ldap_username] [ldap_password]

team_name         name of the team that will use the service user
cluster_name      name of the cluster that the user will be used
ldap_username     valid ldap username (for putting kubeconfig into Vault)
ldap_password     valid ldap password (for putting kubeconfig into Vault)

EOF
}

if [[ "${1}" == "-h" || "${1}" == "--help"  || "${#}" -lt "4" ]]; then print_help && exit 0; fi

# create service account, and get token from secret
kubectl create serviceaccount serviceuser-${TEAMNAME}
TOKEN=$(kubectl get secret $(kubectl get serviceaccount serviceuser-${TEAMNAME} -o=jsonpath='{.secrets[0].name}') -o=jsonpath='{.data.token}' | base64 --decode)

# generate a valid JSON kubeconfig with service account token and cluster info
TMP_KUBECONFIG=$(mktemp)
kubectl config --kubeconfig=${TMP_KUBECONFIG} set-credentials service-user --token=${TOKEN}
kubectl config --kubeconfig=${TMP_KUBECONFIG} set-cluster ${CLUSTER_NAME} --server=https://apiserver.${CLUSTER_NAME}.nais.io^
kubectl config --kubeconfig=${TMP_KUBECONFIG} set-context ${CLUSTER_NAME} --user=service-user
kubectl config --kubeconfig=${TMP_KUBECONFIG} use-context ${CLUSTER_NAME}
JSON_KUBECONFIG=$(mktemp)
kubectl config --kubeconfig=${TMP_KUBECONFIG} view -o=json > ${JSON_KUBECONFIG}

# wrap kubeconfig and output unwrap token
VAULT_TOKEN=$(curl -s -XPOST -d "{\"password\": \"${LDAP_PASSWORD}\"}" https://vault.adeo.no/v1/auth/ldap/login/${LDAP_USERNAME} | jq .auth.client_token | tr -d '"')
UNWRAP_TOKEN=$(curl -s -XPOST -H "X-Vault-Token: ${VAULT_TOKEN}" -H "X-Vault-Wrap-TTL: 24h" --data @${JSON_KUBECONFIG} https://vault.adeo.no/v1/sys/wrapping/wrap | jq .wrap_info.token | tr -d '"')
echo "To get a working kubeconfig for your service account, go to https://vault.adeo.no/ui/vault/tools/unwrap and enter this unwrap token:"
echo "${UNWRAP_TOKEN}"
