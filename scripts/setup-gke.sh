#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID=$(gcloud config get-value project)
REGION="us-central1"
CLUSTER="tmdb-cluster"
NAMESPACE="tmdb"
KSA="tmdb-ksa"
NODE_SA="tmdb-gke-node-sa@${PROJECT_ID}.iam.gserviceaccount.com"
CLOUDSQL_SA="tmdb-cloudsql-sa@${PROJECT_ID}.iam.gserviceaccount.com"

set -a
source .env
set +a

DB_PASSWORD=$(gcloud secrets versions access latest --secret=tmdb-db-password)

echo "==> GKE cluster..."
if gcloud container clusters describe "$CLUSTER" --region="$REGION" >/dev/null 2>&1; then
  echo "    already exists, skipping create"
else
  gcloud container clusters create-auto "$CLUSTER" \
    --region="$REGION" \
    --service-account="$NODE_SA" \
    --enable-secret-manager
fi

gcloud container clusters get-credentials "$CLUSTER" --region="$REGION"

echo "==> Namespace + Workload Identity..."
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl create serviceaccount "$KSA" -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl annotate serviceaccount "$KSA" -n "$NAMESPACE" \
  iam.gke.io/gcp-service-account="$CLOUDSQL_SA" \
  --overwrite

echo "==> Syncing Secret Manager values (tmdb-config is now created/synced by the Secret Manager CSI driver)..."
put_secret() {
  local name="$1" value="$2"
  if gcloud secrets describe "$name" >/dev/null 2>&1; then
    printf '%s' "$value" | gcloud secrets versions add "$name" --data-file=-
  else
    printf '%s' "$value" | gcloud secrets create "$name" --data-file=-
  fi
}

put_secret tmdb-api-key "$TMDB_API_KEY"
put_secret tmdb-go-database-url "postgres://postgres:${DB_PASSWORD}@localhost:5432/tmdb?sslmode=disable"
put_secret tmdb-instance-connection-name "${PROJECT_ID}:${REGION}:tmdb-postgres"

echo "==> Triggering an initial deploy..."
gcloud builds triggers run deploy-on-main --branch=main --region="$REGION"

echo "Done. Watch it with: gcloud builds list --ongoing"
