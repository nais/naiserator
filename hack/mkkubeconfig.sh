#!/bin/bash

set -o errexit
set -o pipefail

print() {
    echo '<.>' $* >&2
}

help() {
    echo
    echo "mkkubeconfig: generate certificate, key and kubeconfig for a machine user."
    echo
    echo "This command will connect to your current Kubernetes cluster, and create"
    echo "a certificate for authentication with the API server. Role bindings will"
    echo "have to be created separately."
    echo
    echo "Syntax: mkkubeconfig <SUBJECT>"
    echo
    exit 1
}

KEY_TYPE="rsa:2048"

subject=$1
if [ "$subject" == "" ] || [ "$subject" == "--help" ] || [ "$subject" == "-h" ]; then
  help
fi

kubeconfig=`mktemp`
current_context=`kubectl config current-context`

for cluster_name in preprod-fss preprod-sbs prod-fss prod-sbs; do

  kubectl config set current-context $cluster_name >&2

  cluster_name=`kubectl config view --minify -o=jsonpath='{.clusters[0].name}'`
  api_server_url=`kubectl config view --minify -o=jsonpath='{.clusters[0].cluster.server}'`
  credentials_name="${cluster_name}-${subject}"

  print Creating a TLS certificate in ${cluster_name} with CN="${subject}"
  print Kubernetes API server is ${api_server_url}

  key=`mktemp`
  cert=`mktemp`
  csr=`mktemp`
  kubecsr=`mktemp`

  print Creating a new private key and certificate signing request
  openssl req -new -newkey $KEY_TYPE -keyout $key -nodes -out $csr -subj "/CN=${subject}" >&2

  print Request certificate signing through a Kubernetes resource
  cat > $kubecsr <<EOF
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: ${subject}
spec:
  groups:
    - system:authenticated
  request: $(base64 < ${csr} | tr -d '\n')
  usages:
    - digital signature
    - key encipherment
    - client auth
EOF

  kubectl create -n default -f $kubecsr >&2

  print Signing the certificate using the Kubernetes CA
  kubectl certificate approve $subject -n default >&2

  print Downloading the signed certificate and delete it from Kubernetes
  kubectl get csr $subject -o jsonpath='{.status.certificate}' | base64 --decode > $cert
  kubectl delete csr $subject >&2

  kubectl --kubeconfig $kubeconfig config set-cluster $cluster_name --server $api_server_url --insecure-skip-tls-verify >&2
  kubectl --kubeconfig $kubeconfig config set-credentials $credentials_name --client-key $key --client-certificate $cert --embed-certs >&2
  kubectl --kubeconfig $kubeconfig config set-context $cluster_name --cluster $cluster_name --user $credentials_name --namespace default >&2
  kubectl --kubeconfig $kubeconfig config set current-context $cluster_name >&2

  rm -f $key $cert $csr $kubecsr
done

kubectl config set-context $current_context >&2

print
print Your kubeconfig is ready:
print
echo '# vi: se ft=yaml:'
echo '---'
cat $kubeconfig
