FROM oneflow-dashboard-web:templates-ux-20260907
COPY runtime/ /app/
COPY assets/static/ /app/.next/static/
