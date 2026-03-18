#!/bin/sh
set -e

# Start the application, passing through all arguments
exec omcgo-app "$@"
