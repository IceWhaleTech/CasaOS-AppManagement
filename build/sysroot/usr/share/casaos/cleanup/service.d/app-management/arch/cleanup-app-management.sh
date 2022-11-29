#!/bin/bash

set -e

readonly CASA_SERVICES=(
    "casaos-app-management.service"
)

readonly CASA_EXEC=casaos-app-management
readonly CASA_CONF=/etc/casaos/app-management.conf
readonly CASA_DB=/var/lib/casaos/db/app-management.db

readonly aCOLOUR=(
    '\e[38;5;154m' # green  	| Lines, bullets and separators
    '\e[1m'        # Bold white	| Main descriptions
    '\e[90m'       # Grey		| Credits
    '\e[91m'       # Red		| Update notifications Alert
    '\e[33m'       # Yellow		| Emphasis
)

Show() {
    # OK
    if (($1 == 0)); then
        echo -e "${aCOLOUR[2]}[$COLOUR_RESET${aCOLOUR[0]}  OK  $COLOUR_RESET${aCOLOUR[2]}]$COLOUR_RESET $2"
    # FAILED
    elif (($1 == 1)); then
        echo -e "${aCOLOUR[2]}[$COLOUR_RESET${aCOLOUR[3]}FAILED$COLOUR_RESET${aCOLOUR[2]}]$COLOUR_RESET $2"
    # INFO
    elif (($1 == 2)); then
        echo -e "${aCOLOUR[2]}[$COLOUR_RESET${aCOLOUR[0]} INFO $COLOUR_RESET${aCOLOUR[2]}]$COLOUR_RESET $2"
    # NOTICE
    elif (($1 == 3)); then
        echo -e "${aCOLOUR[2]}[$COLOUR_RESET${aCOLOUR[4]}NOTICE$COLOUR_RESET${aCOLOUR[2]}]$COLOUR_RESET $2"
    fi
}

Warn() {
    echo -e "${aCOLOUR[3]}$1$COLOUR_RESET"
}

trap 'onCtrlC' INT
onCtrlC() {
    echo -e "${COLOUR_RESET}"
    exit 1
}

if [[ ! -x "$(command -v ${CASA_EXEC})" ]]; then
    Show 2 "${CASA_EXEC} is not detected, exit the script."
    exit 1
fi

while true; do
    echo -n -e "         ${aCOLOUR[4]}Do you want delete app management database? Y/n :${COLOUR_RESET}"
    read -r input
    case $input in
    [yY][eE][sS] | [yY])
        REMOVE_APP_MANAGEMENT_DATABASE=true
        break
        ;;
    [nN][oO] | [nN])
        REMOVE_APP_MANAGEMENT_DATABASE=false
        break
        ;;
    *)
        Warn "         Invalid input..."
        ;;
    esac
done

for SERVICE in "${CASA_SERVICES[@]}"; do
    Show 2 "Stopping ${SERVICE}..."
    systemctl disable --now "${SERVICE}" || Show 3 "Failed to disable ${SERVICE}"
done

rm -rvf "$(which ${CASA_EXEC})" || Show 3 "Failed to remove ${CASA_EXEC}"
rm -rvf "${CASA_CONF}" || Show 3 "Failed to remove ${CASA_CONF}"

if [[ ${REMOVE_APP_MANAGEMENT_DATABASE} == true ]]; then
    rm -rvf "${CASA_DB}" || Show 3 "Failed to remove ${CASA_DB}"
fi
