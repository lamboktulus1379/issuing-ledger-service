#!/usr/bin/env bash

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONNECT_URL="${CONNECT_URL:-http://localhost:8083}"
CONNECTOR_FILE="${CONNECTOR_FILE:-${PROJECT_ROOT}/infrastructure/cdc/outbox-connector.json}"
CONNECTOR_NAME="${CONNECTOR_NAME:-issuing-outbox-connector}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-60}"
RETRY_SECONDS="${RETRY_SECONDS:-2}"

if [[ ! -f "${CONNECTOR_FILE}" ]]; then
	echo "Connector configuration not found: ${CONNECTOR_FILE}" >&2
	exit 1
fi

echo "Waiting for Kafka Connect at ${CONNECT_URL}..."

# Kafka Connect runs a JVM and commonly needs 30-60 seconds before its REST
# listener accepts requests. Retry rather than failing a Compose startup race.
for ((attempt = 1; attempt <= MAX_ATTEMPTS; attempt++)); do
	if curl --fail --silent --show-error --max-time 5 "${CONNECT_URL}/" >/dev/null 2>&1; then
		echo "Kafka Connect is healthy after attempt ${attempt}."
		break
	fi

	if (( attempt == MAX_ATTEMPTS )); then
		echo "Kafka Connect did not become ready after $((MAX_ATTEMPTS * RETRY_SECONDS)) seconds." >&2
		exit 1
	fi

	echo "Kafka Connect is not ready; retrying in ${RETRY_SECONDS}s (${attempt}/${MAX_ATTEMPTS})..."
	sleep "${RETRY_SECONDS}"
done

response_file="$(mktemp)"
trap 'rm -f "${response_file}"' EXIT

http_code="$(curl --silent --show-error --output "${response_file}" --write-out '%{http_code}' \
	-X POST "${CONNECT_URL}/connectors" \
	-H 'Content-Type: application/json' \
	--data-binary "@${CONNECTOR_FILE}")"

case "${http_code}" in
	201|202)
		echo "Connector ${CONNECTOR_NAME} registered successfully."
		cat "${response_file}"
		echo
		;;
	409)
		echo "Connector ${CONNECTOR_NAME} already exists; registration is idempotent."
		if ! curl --fail --silent --show-error "${CONNECT_URL}/connectors/${CONNECTOR_NAME}"; then
			echo "Existing connector could not be read back." >&2
			exit 1
		fi
		echo
		;;
	*)
		echo "Connector registration failed with HTTP ${http_code}:" >&2
		cat "${response_file}" >&2
		exit 1
		;;
esac