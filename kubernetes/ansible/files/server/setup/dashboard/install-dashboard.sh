#!/bin/bash
LOG_FILE=install.log
NS=kubernetes-dashboard
function clean_dashboard() {
  kubectl delete ns "$NS"  >> "$LOG_FILE"
  kubectl delete clusterrolebinding "$NS"  >> "$LOG_FILE"
  kubectl delete clusterrole "$NS"  >> "$LOG_FILE"
  kubectl clusterrolebinding admin-user >> "$LOG_FILE"
}

[[ -f "$LOG_FILE" ]] && rm "$LOG_FILE"
[[ $(kubectl get namespace | grep "$NS") ]]; clean_dashboard   >> "$LOG_FILE"

GITHUB_URL=https://github.com/kubernetes/dashboard/releases
VERSION_KUBE_DASHBOARD=$(curl -w '%{url_effective}' -I -L -s -S ${GITHUB_URL}/latest -o /dev/null | sed -e 's|.*/||')
kubectl create -f  https://raw.githubusercontent.com/kubernetes/dashboard/${VERSION_KUBE_DASHBOARD}/aio/deploy/recommended.yaml   >> "$LOG_FILE"
kubectl create -f admin-user.yml -f admin-user-role.yml   >> "$LOG_FILE"

PROXY=/usr/local/bin/start-proxy
cat <<EOF > "$PROXY"
  echo "URL to server is http://localhost:8001/api/v1/namespaces/$NS/services/https:$NS:/proxy/"
  kubectl -n "$NS" describe secret admin-user-token | grep '^token'
  kubectl proxy
EOF
chmod 755 "$PROXY"

