# Migrating Backend to Google Cloud Run

## When to migrate

Railway free tier gives 500 hours/month — enough for ~20 days of continuous uptime.
Migrate when:
- You need always-on (no cold starts)
- Monthly requests exceed Railway Starter plan value ($5/mo)
- You want cost-per-request billing (~$0–3/mo at low personal traffic)

## Cost comparison

| Service | Always-on | Cost |
|---------|-----------|------|
| Railway free | No (sleeps) | $0 (500h/mo) |
| Railway Starter | Yes | $5/mo |
| Cloud Run | No (scales to 0) | ~$0–3/mo at low traffic |
| Cloud Run min-instances=1 | Yes | ~$7/mo (us-central1) |

Cloud Run billed only on requests — ideal for low-traffic personal apps.

## Prerequisites

```bash
brew install google-cloud-sdk
gcloud auth login
gcloud config set project YOUR_PROJECT_ID
gcloud services enable run.googleapis.com artifactregistry.googleapis.com
```

## Build and push Docker image

```bash
# From repo root
gcloud artifacts repositories create mylifeos \
  --repository-format=docker \
  --location=us-central1

docker build -t us-central1-docker.pkg.dev/YOUR_PROJECT_ID/mylifeos/backend:latest ./backend
docker push us-central1-docker.pkg.dev/YOUR_PROJECT_ID/mylifeos/backend:latest
```

## Deploy to Cloud Run

```bash
gcloud run deploy mylifeos-backend \
  --image us-central1-docker.pkg.dev/YOUR_PROJECT_ID/mylifeos/backend:latest \
  --region us-central1 \
  --platform managed \
  --allow-unauthenticated \
  --port 8080 \
  --set-env-vars "ENV=production,SUPABASE_URL=...,SUPABASE_JWT_SECRET=...,DATABASE_URL=..."
```

## Update frontend to point to Cloud Run

In Vercel → Environment Variables, update:

```
VITE_API_URL = https://mylifeos-backend-xxxx-uc.a.run.app/api/v1
```

Redeploy Vercel (push an empty commit or trigger manually).

## Set up CI/CD for Cloud Run

Add to `.github/workflows/deploy.yml`:

```yaml
deploy-backend:
  runs-on: ubuntu-latest
  if: github.ref == 'refs/heads/main'
  steps:
    - uses: actions/checkout@v4
    - uses: google-github-actions/auth@v2
      with:
        credentials_json: ${{ secrets.GCP_SA_KEY }}
    - uses: google-github-actions/deploy-cloudrun@v2
      with:
        service: mylifeos-backend
        region: us-central1
        image: us-central1-docker.pkg.dev/${{ vars.GCP_PROJECT_ID }}/mylifeos/backend:latest
```

Create a GCP Service Account with `Cloud Run Admin` + `Artifact Registry Writer` roles.
Add the JSON key as `GCP_SA_KEY` secret in GitHub repo settings.

## Disable Railway

Once Cloud Run is verified working, delete the Railway service to avoid wasted hours.

## No backend code changes needed

The same `Dockerfile` and environment variables work identically on Cloud Run.
The only change is where the image runs.
