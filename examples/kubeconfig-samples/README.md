# Kubeconfig Sample Files

This directory contains example kubeconfig configurations for testing and reference.

## Files

- `development.yaml` - Single cluster development configuration
- `multi-cluster.yaml` - Multi-cluster setup (dev/staging/prod)
- `local-minikube.yaml` - Local Minikube configuration example

## Usage

These files are **examples only** and contain placeholder data. To use them:

1. Copy a sample file:
   ```bash
   cp examples/kubeconfig-samples/multi-cluster.yaml ~/my-kubeconfig.yaml
   ```

2. Edit with your actual cluster details:
   - Replace `server` URLs with your actual API server addresses
   - Add your actual certificate data
   - Update cluster names and contexts

3. Test with Kopilot:
   ```bash
   KUBECONFIG=~/my-kubeconfig.yaml kopilot
   ```

## Security Notes

⚠️ **Never commit real kubeconfig files with actual credentials!**

- These examples use placeholder/omitted data
- Real kubeconfig files contain sensitive credentials
- Always use `.gitignore` to exclude real kubeconfig files
- Consider using separate authentication methods in production

## Creating Your Own

To create a kubeconfig for your cluster:

```bash
# For GKE
gcloud container clusters get-credentials CLUSTER_NAME --region=REGION

# For EKS
aws eks update-kubeconfig --name CLUSTER_NAME --region REGION

# For AKS
az aks get-credentials --resource-group RESOURCE_GROUP --name CLUSTER_NAME

# For Minikube
minikube start
# Kubeconfig is automatically created at ~/.kube/config
```

## Merging Multiple Kubeconfigs

```bash
# Set KUBECONFIG to multiple files
export KUBECONFIG=~/.kube/config:~/.kube/config-dev:~/.kube/config-prod

# View merged config
kubectl config view --merge --flatten

# Save merged config
kubectl config view --merge --flatten > ~/.kube/config-merged
```

## See Also

- [Kubernetes Documentation: Organizing Cluster Access](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/)
- [kubectl config reference](https://kubernetes.io/docs/reference/kubectl/cheatsheet/#kubectl-context-and-configuration)
