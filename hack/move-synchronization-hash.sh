#!/bin/bash -e
#
# Move synchronization hash field from annotations to status.

namespaces=`kubectl get namespace -o jsonpath='{.items[*].metadata.name}'`

for n in $namespaces; do
    echo "=== namespace: $n ==="
    apps=`kubectl get app --namespace $n -o jsonpath='{.items[*].metadata.name}'`
    for a in $apps; do
        hash=`kubectl get app --namespace $n $a -o jsonpath="{.metadata.annotations.nais\.io/lastSyncedHash}"`
        patch="{\"status\":{\"synchronizationHash\":\"$hash\"}}"
        kubectl patch app $a --namespace $n --type=merge --patch $patch
    done
done
