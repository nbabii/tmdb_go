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
    --service-account="$NODE_SA"
fi

gcloud container clusters get-credentials "$CLUSTER" --region="$REGION"

echo "==> Namespace + Workload Identity..."
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl create serviceaccount "$KSA" -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl annotate serviceaccount "$KSA" -n "$NAMESPACE" \
  iam.gke.io/gcp-service-account="$CLOUDSQL_SA" \
  --overwrite

echo "==> tmdb-config Secret..."
kubectl create secret generic tmdb-config -n "$NAMESPACE" \
  --from-literal=TMDB_API_KEY="$TMDB_API_KEY" \
  --from-literal=TMDB_BASE_URL="$TMDB_BASE_URL" \
  --from-literal=GO_DATABASE_URL="postgres://postgres:${DB_PASSWORD}@localhost:5432/tmdb?sslmode=disable" \
  --from-literal=PORT="${PORT:-8088}" \
  --from-literal=INSTANCE_CONNECTION_NAME="${PROJECT_ID}:${REGION}:tmdb-postgres" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "==> Triggering an initial deploy..."
gcloud builds triggers run deploy-on-main --branch=main --region="$REGION"

echo "Done. Watch it with: gcloud builds list --ongoing"
