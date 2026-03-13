#!/bin/sh
set -e

# Run database migrations before starting the application
if [ -n "$OMCGO_DB_DSN" ]; then
    echo "Running database migrations..."
    omcgo-migrate --dsn "$OMCGO_DB_DSN" --path /etc/omcgo/migrations up
    echo "Database migrations completed."
fi

# Start the application, passing through all arguments
exec omcgo-app "$@"
