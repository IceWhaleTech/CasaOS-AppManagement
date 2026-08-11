#!/bin/bash

set -e

BUILD_PATH=$(dirname "${BASH_SOURCE[0]}")/../../..

readonly BUILD_PATH
readonly APP_NAME_SHORT=app-management

__get_setup_script_directory_by_os_release() {
	local service_directory
	local os_release_file
	local candidate
	local distro
	local candidates=()

	if [[ -n ${CASAOS_SETUP_SERVICE_DIRECTORY:-} ]]; then
		service_directory=$(cd "${CASAOS_SETUP_SERVICE_DIRECTORY}" && pwd -P)
	else
		service_directory=$(cd "$(dirname "${BASH_SOURCE[0]}")/../service.d/${APP_NAME_SHORT}" && pwd -P)
	fi
	os_release_file=${CASAOS_OS_RELEASE_FILE:-/etc/os-release}

	if [[ ! -r "${os_release_file}" ]]; then
		echo "Unsupported OS: unable to read ${os_release_file}" >&2
		return 1
	fi

	# shellcheck source=/dev/null
	source "${os_release_file}"

	if [[ -n ${ID:-} && -n ${VERSION_CODENAME:-} ]]; then
		candidates+=("${ID}/${VERSION_CODENAME}")
	fi
	if [[ -n ${ID:-} ]]; then
		candidates+=("${ID}")
	fi
	for distro in ${ID_LIKE:-}; do
		if [[ -n ${VERSION_CODENAME:-} ]]; then
			candidates+=("${distro}/${VERSION_CODENAME}")
		fi
		candidates+=("${distro}")
	done

	for candidate in "${candidates[@]}"; do
		if [[ -f "${service_directory}/${candidate}/setup-${APP_NAME_SHORT}.sh" ]]; then
			cd "${service_directory}/${candidate}"
			pwd -P
			return 0
		fi
	done

	echo "Unsupported OS: ${ID:-unknown} ${VERSION_CODENAME:-unknown} (${ID_LIKE:-})" >&2
	return 1
}

SETUP_SCRIPT_DIRECTORY=$(__get_setup_script_directory_by_os_release)

readonly SETUP_SCRIPT_DIRECTORY
readonly SETUP_SCRIPT_FILENAME="setup-${APP_NAME_SHORT}.sh"
readonly SETUP_SCRIPT_FILEPATH="${SETUP_SCRIPT_DIRECTORY}/${SETUP_SCRIPT_FILENAME}"

{
    echo "🟩 Running ${SETUP_SCRIPT_FILENAME}..."
    $BASH "${SETUP_SCRIPT_FILEPATH}" "${BUILD_PATH}"
} || {
    echo "🟥 ${SETUP_SCRIPT_FILENAME} failed."
    exit 1
}

echo "✅ ${SETUP_SCRIPT_FILENAME} finished."
