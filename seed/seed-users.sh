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

  user_ref=$(psql "$DATABASE_URL" -t -A -c "SELECT user_ref FROM users WHERE email = '$email'")

  echo "Seeded user: $username ($email) as $role (user_ref: $user_ref)"

  if [ -n "$BOOKINGS_DATABASE_URL" ] && [ -n "$user_ref" ]; then
    psql "$BOOKINGS_DATABASE_URL" -v ON_ERROR_STOP=1 -c "
      INSERT INTO users (userref) VALUES ('$user_ref') ON CONFLICT DO NOTHING;
    "
    echo "  -> added to bookings.users"
  fi
done
