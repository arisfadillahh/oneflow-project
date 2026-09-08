#!/usr/bin/env sh
set -eu

deploy_dir=/home/goffath/oneflow-deploy
stamp=20260908-inbox-i18n
compose="$deploy_dir/docker-compose.yml"

cd "$deploy_dir"
df -h /
docker exec oneflow-postgres psql -U postgres -d oneflow -Atc \
  'SELECT pg_size_pretty(pg_database_size(current_database()));'

cp "$compose" "$deploy_dir/docker-compose.before-$stamp.yml"
docker exec oneflow-postgres pg_dump -U postgres -d oneflow -Fc \
  > "$deploy_dir/oneflow-before-$stamp.dump"

cp "$compose" "$compose.next"
sed -i 's#image: oneflow-app-backend:inbox-fix2-20260907#image: oneflow-app-backend:inbox-i18n-20260908#' "$compose.next"
sed -i 's#image: oneflow-dashboard-web:templates-page-v6-20260908#image: oneflow-dashboard-web:inbox-i18n-20260908#' "$compose.next"
grep -q 'oneflow-app-backend:inbox-i18n-20260908' "$compose.next"
grep -q 'oneflow-dashboard-web:inbox-i18n-20260908' "$compose.next"
mv "$compose.next" "$compose"

wait_healthy() {
  container="$1"
  attempt=0
  while [ "$attempt" -lt 30 ]; do
    status=$(docker inspect "$container" --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}')
    [ "$status" = healthy ] && return 0
    attempt=$((attempt + 1))
    sleep 2
  done
  echo "$container failed health check: $status" >&2
  return 1
}

docker compose up -d --no-deps app-backend
wait_healthy oneflow-app-backend
docker compose up -d --no-deps dashboard-web
wait_healthy oneflow-dashboard-web

docker inspect oneflow-app-backend oneflow-dashboard-web \
  --format '{{.Name}}|{{.Config.Image}}|{{.Image}}|{{.State.Health.Status}}'
docker exec oneflow-postgres psql -U postgres -d oneflow -Atc \
  "SELECT column_name FROM information_schema.columns WHERE table_name = 'agents' AND column_name = 'preferred_locale'; SELECT to_regclass('public.conversation_reads');"
