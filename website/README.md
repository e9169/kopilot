# Kopilot Website

This directory contains the Jekyll-based website for Kopilot, built using GitHub's native tools.

## Features

- ✅ Built with **Jekyll** (GitHub's native static site generator)
- ✅ Uses **GitHub Pages** theme (Cayman)
- ✅ Automatic deployment via **GitHub Actions**
- ✅ SEO optimized with `jekyll-seo-tag`
- ✅ RSS feed with `jekyll-feed`
- ✅ Fully responsive design
- ✅ Zero external dependencies

## Local Development

### Prerequisites

```bash
# Install Ruby (if not already installed)
brew install ruby

# Add Ruby to PATH
echo 'export PATH="/usr/local/opt/ruby/bin:$PATH"' >> ~/.zshrc
```

### Setup

```bash
# Navigate to the website directory
cd website

# Install dependencies
bundle install

# Serve the site locally
bundle exec jekyll serve --baseurl /kopilot

# Or with live reload
bundle exec jekyll serve --livereload --baseurl /kopilot
```

Visit `http://localhost:4000/kopilot` in your browser.

## Deployment

The site automatically deploys to GitHub Pages when you push to the main branch.

### Manual Deployment Steps

1. **Enable GitHub Pages:**
   - Go to your repository settings
   - Navigate to "Pages" section
   - Source: Select "GitHub Actions"

2. **Push your changes:**
   ```bash
   git add .
   git commit -m "Add Jekyll website"
   git push origin main
   ```

3. **Wait for deployment:**
   - GitHub Actions will automatically build and deploy
   - Check the "Actions" tab for deployment status
   - Site will be available at: `https://e9169.github.io/kopilot`

## File Structure

```
.
├── _config.yml           # Jekyll configuration
├── _layouts/             # Custom layouts
│   └── default.html      # Main layout template
├── index.md              # Homepage
├── docs.md               # Documentation page
├── Gemfile               # Ruby dependencies
└── .github/
    └── workflows/
        └── jekyll.yml    # GitHub Actions workflow
```

## Customization

### Update Site Title/Description

Edit `_config.yml`:
```yaml
title: Your Title
description: Your description
```

### Change Theme

GitHub Pages supports several themes. Edit `_config.yml`:
```yaml
theme: jekyll-theme-minimal
# Or use remote themes
remote_theme: pages-themes/architect@v0.2.0
```

Available themes:
- `jekyll-theme-cayman` (current)
- `jekyll-theme-minimal`
- `jekyll-theme-architect`
- `jekyll-theme-slate`
- `jekyll-theme-hacker`

### Add New Pages

Create a new `.md` file:
```markdown
---
layout: default
title: My Page
permalink: /my-page/
---

# My Page Content
```

## GitHub Actions Workflow

The `.github/workflows/jekyll.yml` file handles automatic deployment:
- Triggers on push to main branch
- Builds the Jekyll site
- Deploys to GitHub Pages
- No manual intervention needed

## Troubleshooting

### Site not showing up

1. Check GitHub Actions logs in the "Actions" tab
2. Verify GitHub Pages is enabled in repository settings
3. Ensure `baseurl` in `_config.yml` matches your repository name

### Local build errors

```bash
# Clear cache and rebuild
bundle exec jekyll clean
bundle exec jekyll build
```

### Update dependencies

```bash
bundle update
```

## Resources

- [Jekyll Documentation](https://jekyllrb.com/docs/)
- [GitHub Pages Documentation](https://docs.github.com/en/pages)
- [Jekyll Themes](https://pages.github.com/themes/)
- [Liquid Template Language](https://shopify.github.io/liquid/)

## License

MIT License - see LICENSE file for details.
