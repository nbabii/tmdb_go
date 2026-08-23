#!/usr/bin/env bash
set -euo pipefail

REGION="us-central1"
CLUSTER="tmdb-cluster"
NAMESPACE="tmdb"

# Delete the namespace (and its LoadBalancer Service) first so the external
# IP is released cleanly, rather than torn down abruptly with the cluster.
kubectl delete namespace "$NAMESPACE" --ignore-not-found

gcloud container clusters delete "$CLUSTER" --region="$REGION" --quiet

echo "Cluster deleted. Cloud SQL and Artifact Registry are untouched and still billing."
