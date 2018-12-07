#!/bin/bash

TEAMNAME="${1}"

print_help() {
  cat <<EOF

This script will create a service user for a team, generate a valid kubeconfig with it's token and put it into Vault.

usage: ./create_serviceuser.sh team cluster [cluster...]

team         name of the team that will use the service user
cluster      name of the cluster(s) that the user will be created in

EOF
}

if [[ "${1}" == "-h" || "${1}" == "--help"  || "${#}" -lt "2" ]]; then print_help && exit 0; fi

shift

for prog in kubectl jq base64 curl; do
    which $prog >/dev/null 2>&1
    if [[ $? -ne 0 ]]; then
        echo "Cannot find '$prog' in path, please install it to continue"
        exit 1
    fi
done

set -e

TMP_KUBECONFIG=$(mktemp)
k="kubectl config --kubeconfig=${TMP_KUBECONFIG}"

current_context=`kubectl config current-context`

for cluster_name in $*; do

    kubectl config set current-context $cluster_name

    cluster_name=`kubectl config view --minify -o=jsonpath='{.clusters[0].name}'`
    api_server_url=`kubectl config view --minify -o=jsonpath='{.clusters[0].cluster.server}'`
    credentials_name="${cluster_name}-${TEAMNAME}"
    serviceaccount="serviceuser-${TEAMNAME}"

    # create service account, and get token from secret
    kubectl --namespace default create serviceaccount "${serviceaccount}"
    TOKEN=$(kubectl --namespace default get secret $(kubectl --namespace default get serviceaccount "${serviceaccount}" -o=jsonpath='{.secrets[0].name}') -o=jsonpath='{.data.token}' | base64 --decode)

    # generate a valid JSON kubeconfig with service account token and cluster info
    ${k} set-cluster $cluster_name --server $api_server_url --insecure-skip-tls-verify >&2
    ${k} set-credentials $credentials_name --token=${TOKEN} >&2
    ${k} set-context $cluster_name --cluster $cluster_name --user $credentials_name --namespace default >&2
    ${k} set current-context $cluster_name >&2

done

kubectl config set-context $current_context

JSON_KUBECONFIG=$(mktemp)
${k} view -o=json > ${JSON_KUBECONFIG}

cat ${JSON_KUBECONFIG}

read -p "Enter LDAP username: " LDAP_USERNAME
read -sp "Enter LDAP password: " LDAP_PASSWORD

# wrap kubeconfig and output unwrap token
VAULT_TOKEN=$(curl -s -XPOST -d "{\"password\": \"${LDAP_PASSWORD}\"}" https://vault.adeo.no/v1/auth/ldap/login/${LDAP_USERNAME} | jq .auth.client_token | tr -d '"')
UNWRAP_TOKEN=$(curl -s -XPOST -H "X-Vault-Token: ${VAULT_TOKEN}" -H "X-Vault-Wrap-TTL: 24h" --data @${JSON_KUBECONFIG} https://vault.adeo.no/v1/sys/wrapping/wrap | jq .wrap_info.token | tr -d '"')
echo "To get a working kubeconfig for your service account, go to https://vault.adeo.no/ui/vault/tools/unwrap and enter the following unwrap token (note: only usable once, and will be deleted after 24 hours):"
echo "${UNWRAP_TOKEN}"

rm -f ${TMP_KUBECONFIG} ${JSON_KUBECONFIG}
