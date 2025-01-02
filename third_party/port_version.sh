#!/bin/sh

set -o errexit
set -o pipefail

#readonly DEBUG=1
readonly GREP_BIN=/usr/bin/grep
readonly LESS_BIN=/usr/bin/less
readonly MAKE_BIN=/usr/bin/make
readonly SED_BIN=/usr/bin/sed
readonly SORT_BIN=/usr/bin/sort
readonly LOCALBASE=/usr/local

PORTSDIR=$(${MAKE_BIN} -C / -V PORTSDIR)
# INDEXFILE=$(${MAKE_BIN} -C "${PORTSDIR}" -V INDEXFILE)
readonly GIT_CMD="${LOCALBASE}/bin/git --no-pager -C ${PORTSDIR}"

if [ -n "${DEBUG}" ]; then
	echo "PORTSDIR=${PORTSDIR}"
	# echo "INDEXFILE=${INDEXFILE}"
	echo "GIT_CMD=${GIT_CMD}"
fi

COMMIT_ID="${1}"
if [ -z "${COMMIT_ID}" ]; then
	COMMIT_ID=$(${GIT_CMD} rev-parse --short HEAD)
	test -z "${DEBUG}" || echo "COMMIT_ID=${COMMIT_ID}"
	${GIT_CMD} reset --hard --no-recurse-submodule
	${GIT_CMD} pull --all --prune --stat --verbose
fi

exec ${GIT_CMD} diff --no-color --name-only "${COMMIT_ID}" |
	${SED_BIN} -rn 's,(.*/.*)/[^/]+$,\1,p' |
	${SORT_BIN} --unique |
	${HOME}/bin/portsindexup ${DEBUG:+-verbose} &&
	echo &&
	${LOCALBASE}/sbin/pkg-static version --verbose --not-like = |
	${GREP_BIN} --invert-match --regexp '[[:space:]]>[[:space:]]' --regexp '[[:space:]]orphaned:[[:space:]]local/' |
	${LESS_BIN} --no-init --quit-if-one-screen &&
	echo
