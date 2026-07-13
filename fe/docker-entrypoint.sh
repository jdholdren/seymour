#!/bin/sh
cat <<EOF > /usr/share/nginx/html/config.js
window.__CONFIG__ = {
  VITE_API_HOST: "${VITE_API_HOST:-}"
};
EOF
exec "$@"
