#!/bin/sh
set -e

SEED_FILE="/seed/users.csv"

tail -n +2 "$SEED_FILE" | while IFS=',' read -r username email role; do
  [ -z "$username" ] && continue
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "
    INSERT INTO users (username, email, role_id)
    SELECT '$username', '$email', r.id
    FROM roles r WHERE r.name = '$role'
    ON CONFLICT (email) DO NOTHING;
  "
  echo "Seeded user: $username ($email) as $role"
done
