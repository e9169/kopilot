---
layout: default
title: Home
---

<div style="text-align: center; padding: 4rem 0;">
  <h1 style="font-size: 3rem; margin-bottom: 1rem;">THE AI THAT MANAGES YOUR KUBERNETES</h1>
  <p style="font-size: 1.25rem; color: #666; margin-bottom: 2rem;">Deploy services, scale pods, debug issues, manage configs.<br>All from your terminal.</p>
  <a href="https://github.com/{{ site.repository }}" class="btn" style="background: #28a745; color: white; padding: 1rem 2rem; text-decoration: none; border-radius: 0.5rem; display: inline-block; margin: 0.5rem;">Get Started</a>
  <a href="/kopilot/docs" class="btn" style="background: #6c757d; color: white; padding: 1rem 2rem; text-decoration: none; border-radius: 0.5rem; display: inline-block; margin: 0.5rem;">View Docs</a>
</div>

---

## ⟩ Quick Start

First, install GitHub Copilot CLI (choose one):

```bash
# Option A: Using npm
npm install -g @githubnext/github-copilot-cli
copilot auth login

# Option B: Using GitHub CLI extension
gh extension install github/gh-copilot

# Option C: Using Homebrew
brew install github/gh-copilot/gh-copilot
```

Then build Kopilot from source:

```bash
git clone https://github.com/{{ site.repository }}.git
cd kopilot
make deps
make build
./bin/kopilot
```

---

## ⟩ Features

<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 2rem; margin: 3rem 0;">
  <div style="padding: 1.5rem; border: 1px solid #e1e4e8; border-radius: 0.5rem;">
    <h3>� Cluster Status</h3>
    <p>View all Kubernetes clusters from your kubeconfig with detailed status information.</p>
  </div>
  
  <div style="padding: 1.5rem; border: 1px solid #e1e4e8; border-radius: 0.5rem;">
    <h3>⚖️ Compare Clusters</h3>
    <p>Side-by-side comparison of multiple clusters to spot differences quickly.</p>
  </div>
  
  <div style="padding: 1.5rem; border: 1px solid #e1e4e8; border-radius: 0.5rem;">
    <h3>🏥 Health Monitoring</h3>
    <p>Real-time node and pod health tracking across all your clusters.</p>
  </div>
  
  <div style="padding: 1.5rem; border: 1px solid #e1e4e8; border-radius: 0.5rem;">
    <h3>⚡ Parallel Execution</h3>
    <p>Check all clusters simultaneously for 5-10x faster results.</p>
  </div>
  
  <div style="padding: 1.5rem; border: 1px solid #e1e4e8; border-radius: 0.5rem;">
    <h3>🛠️ kubectl Integration</h3>
    <p>Execute kubectl commands through natural language with interactive confirmations.</p>
  </div>
  
  <div style="padding: 1.5rem; border: 1px solid #e1e4e8; border-radius: 0.5rem;">
    <h3>🔐 Safe by Default</h3>
    <p>Read-only mode protects against accidental changes. Your kubeconfig stays local.</p>
  </div>
  
  <div style="padding: 1.5rem; border: 1px solid #e1e4e8; border-radius: 0.5rem;">
    <h3>💰 Smart Model Selection</h3>
    <p>Automatically switches between GPT-4o-mini for simple queries and GPT-4o for complex tasks, reducing costs by 50-70%.</p>
  </div>
  
  <div style="padding: 1.5rem; border: 1px solid #e1e4e8; border-radius: 0.5rem;">
    <h3>🤖 GitHub Copilot SDK</h3>
    <p>Built with the official GitHub Copilot SDK for natural language interaction with your clusters.</p>
  </div>
</div>

---

## ⟩ Works With Everything

<div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 1rem; margin: 2rem 0;">
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">Kubernetes</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">Helm</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">kubectl</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">Docker</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">Prometheus</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">Grafana</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">ArgoCD</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">Flux</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">Istio</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">AWS EKS</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">GCP GKE</div>
  <div style="padding: 1rem; border: 1px solid #e1e4e8; border-radius: 0.5rem; text-align: center; font-weight: 600;">Azure AKS</div>
</div>

---

## ⟩ Get Started

Ready to supercharge your Kubernetes workflow?

1. **[Install Kopilot →](https://github.com/{{ site.repository }}#installation)**
2. **[Read the Docs →](/kopilot/docs)**
3. **[Join the Community →](https://github.com/{{ site.repository }}/discussions)**

<div style="text-align: center; padding: 3rem 0; background: #f6f8fa; margin: 3rem -2rem; border-radius: 0.5rem;">
  <h3>Built with ❤️ for the Kubernetes community</h3>
  <p style="color: #666;">Open source • Powered by GitHub Copilot • Community driven</p>
</div>
