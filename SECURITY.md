# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| dev     | :white_check_mark: |

We are currently in active development. Once we reach our first stable release, we will maintain security updates for the latest stable version.

## Security Considerations

### Kubeconfig Access

Kopilot requires access to your Kubernetes configuration file (`~/.kube/config` or path specified by `KUBECONFIG` environment variable) to:
- List available clusters
- Query cluster status
- Execute kubectl commands

**Important Security Notes:**

1. **Credentials Access**: Kopilot has full access to all Kubernetes credentials in your kubeconfig
2. **Production Clusters**: Use read-only mode (default) when working with production clusters
3. **Service Accounts**: Consider using limited service accounts for kopilot rather than admin credentials
4. **Audit Logging**: Enable Kubernetes audit logging to track all operations performed through kopilot

### Execution Modes

Kopilot implements two security modes:

#### 🔒 Read-Only Mode (Default)
- **Blocks all write operations** (scale, delete, apply, patch, etc.)
- Only allows read operations (get, describe, logs, etc.)
- Recommended for production cluster access
- Prevents accidental modifications

#### 🔓 Interactive Mode
- Requires **explicit confirmation** for write operations
- Shows exact command before execution
- User can cancel operations
- Use only when write access is needed

**Never use `--interactive` mode** with untrusted input or automated scripts against production clusters.

### AI-Powered Operations

Kopilot uses GitHub Copilot AI to interpret natural language commands:

- **Command Interpretation**: AI may misinterpret instructions
- **Validation Required**: Always review the displayed kubectl command before confirming
- **Production Safety**: Use read-only mode for production clusters
- **Blast Radius**: Limit cluster access to minimize potential damage

### GitHub Copilot Authentication

Kopilot requires GitHub Copilot CLI to be installed and authenticated:

- Uses your GitHub Copilot subscription
- Requires GitHub authentication token
- Commands and cluster data may be sent to GitHub Copilot for processing
- Review [GitHub Copilot Privacy Statement](https://docs.github.com/en/site-policy/privacy-policies/github-copilot-privacy-statement)

**Data Sent to GitHub Copilot:**
- Natural language queries
- Kubectl command outputs
- Cluster resource information
- Error messages

**Not Sent:**
- Kubeconfig files
- Kubernetes credentials
- Cluster certificates

### Environment Variables

Kopilot respects the following environment variables:

- `KUBECONFIG` - Path to kubeconfig file (has access to all credentials in the file)
- `KOPILOT_MODEL_COST_EFFECTIVE` - Override default AI model for simple queries
- `KOPILOT_MODEL_PREMIUM` - Override default AI model for complex operations

### Recommendations

1. **Principle of Least Privilege**
   - Use service accounts with minimal required permissions
   - Create separate kubeconfig files for different environments
   - Restrict kopilot to development/staging clusters when possible

2. **Audit and Monitoring**
   - Enable Kubernetes audit logging
   - Review kubectl operations regularly
   - Monitor for unexpected cluster changes

3. **Network Security**
   - Run kopilot in a secure network environment
   - Use VPN when accessing remote clusters
   - Consider network policies to limit blast radius

4. **Access Control**
   - Don't share kopilot sessions or terminals
   - Use RBAC to limit cluster access
   - Rotate credentials regularly

5. **Code Review**
   - Review kopilot source code before use in sensitive environments
   - Verify the integrity of releases
   - Build from source for maximum security

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue in kopilot, please report it responsibly.

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report security issues via:

1. **Email**: [Create a security advisory on GitHub](https://github.com/e9169/kopilot/security/advisories/new)
2. **GitHub Security Advisory**: Preferred method - allows for private disclosure and coordinated fixes

### What to Include

When reporting a vulnerability, please include:

- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Suggested fix (if you have one)
- Your contact information

### Response Timeline

- **Initial Response**: Within 48 hours
- **Vulnerability Assessment**: Within 1 week
- **Fix Development**: Depends on severity
  - Critical: 1-3 days
  - High: 1-2 weeks
  - Medium: 2-4 weeks
  - Low: Next planned release

### Disclosure Policy

- We will acknowledge your report within 48 hours
- We will provide regular updates on our progress
- We will credit you in the security advisory (unless you prefer to remain anonymous)
- We will coordinate disclosure timing with you
- We will publish a security advisory after the fix is released

## Security Best Practices for Users

### For Development Environments

```bash
# Use read-only mode (default)
kopilot

# Check cluster health without write access
> check all clusters
> show me pods in production
```

### For Operational Tasks

```bash
# Use interactive mode for controlled writes
kopilot --interactive

# Confirm each operation
> scale nginx deployment to 3 replicas
⚠️  Write Operation: kubectl scale deployment nginx --replicas=3
Do you want to proceed? (yes/no): yes
```

### For Production Clusters

1. Create a dedicated read-only service account:
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kopilot-readonly
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kopilot-readonly
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kopilot-readonly
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kopilot-readonly
subjects:
- kind: ServiceAccount
  name: kopilot-readonly
  namespace: default
```

2. Create a dedicated kubeconfig for kopilot:
```bash
# Extract service account token and create kubeconfig
kubectl create token kopilot-readonly > kopilot-token
# ... configure kubeconfig with limited access
export KUBECONFIG=~/.kube/kopilot-readonly.config
kopilot
```

## Dependency Security

Kopilot relies on the following key dependencies:

- **GitHub Copilot SDK**: Official SDK from GitHub
- **Kubernetes Client-Go**: Official Kubernetes Go client
- **kubectl**: Must be installed separately (not bundled)

We regularly update dependencies to address security vulnerabilities. Run `go mod verify` to ensure dependency integrity.

## Security Updates

Security updates will be announced via:
- GitHub Security Advisories
- Release notes
- README.md updates

Subscribe to repository notifications to stay informed about security updates.

## License

This security policy is part of the kopilot project and is covered by the MIT License.
