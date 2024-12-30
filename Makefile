SETENV?=	/usr/bin/env
GO_CMD?=	/usr/local/bin/go
GOFMT?=		/usr/local/bin/gofmt
GOIMPORTS?=	/usr/local/bin/goimports
GOSEC?=		/usr/local/bin/gosec
GIT_BIN?=	/usr/local/bin/git
GIT_CMD?=	${GIT_BIN} --no-pager

COVERAGE_FILE?=	coverage.out
COVERAGE_HTML?=	${COVERAGE_FILE:R}.html

PROJECT_BIN?=	.build/${.CURDIR:T}

MAIN_BRANCH?=	$$(${GIT_CMD} branch --no-color --list main master | sed -rn "s/^\*?[[:space:]]+//p;q")
CURRENT_BRANCH?=$$(${GIT_CMD} branch --no-color --show-current)

SLEEP_SEC?=	7

.sinclude "Makefile.local"

# -s disable symbol table; -w disable DWARF generation
GO_LDFLAGS?=	-s -w -extldflags '-static${WITH_PIE:D-pie}'
.for _i in VERSION PREFIX
. if !empty(_i)
    GO_LDFLAGS+=-X 'main.${_i:tl}=${${_i}}'
. endif
.endfor

.if defined(WITH_PIE)
GO_BUILDFLAGS+= -buildmode=pie
.else
GO_BUILDFLAGS+= -buildmode=exe
.endif

GO_BUILDFLAGS+=	-v -trimpath -buildvcs=false
GO_BUILDFLAGS+=	${GO_LDFLAGS:D-ldflags "${GO_LDFLAGS}"}

#https://stackoverflow.com/questions/64019336/go-compile-to-static-binary-with-pie
#go build  -ldflags '-linkmode external -s -w -extldflags "--static-pie"' -buildmode=pie -tags 'osusergo,netgo,static_build' -o /hello hello.go

.MAIN: all

all: .PHONY build

run: .PHONY .SILENT
	${GO_CMD} run ${GO_BUILDFLAGS} . $$(find environments -name '*.dot')

tidy: .PHONY
	${GO_CMD} mod tidy -v

${PROJECT_BIN}: ${SRCS}
	${SETENV:D${SETENV} GOPROXY=off${GO_ENV:D ${GO_ENV}}} \
		${GO_CMD} build ${GO_BUILDFLAGS} -o ${.TARGET} ${PROJECT}

build: .PHONY ${PROJECT_BIN}	## Build the default binary

fmt: .PHONY	## Format sources
	${GOIMPORTS} -w .
	${GOFMT} -w -s .

test: .PHONY
#	${GO_CMD} test -v -race -covermode=atomic ./...
	${GO_CMD} test -v ./...

clean-all: .PHONY clean
	${GO_CMD} clean -i -r -cache

clean: .PHONY
	rm -f -- ${PROJECT_BIN}

#omit of -covermode=set/count/atomic(w/-race) intentionally
${COVERAGE_FILE}: ${PROFILE_BIN} ${TEST_SRCS}
	${GO_CMD} test -coverprofile=${.TARGET} ./...
	sed -i '' '/_mock\.go:/d' ${.TARGET}

#go tool cover -h
${COVERAGE_HTML}: ${COVERAGE_FILE}
	${GO_CMD} tool cover -html=${.ALLSRC} -o=${.TARGET}

html: .PHONY ${COVERAGE_HTML}	## Show test coverage with HTML
	xdg-open ${.ALLSRC}

#go tool cover -h
cover: .PHONY ${COVERAGE_FILE}	## Show test coverage in percents
	${GO_CMD} tool cover -func=${.ALLSRC}

sleep: .PHONY .SILENT
	sleep ${SLEEP_SEC}

watch: .PHONY .SILENT
	main_branch="${MAIN_BRANCH}"; \
	if [ "_${CURRENT_BRANCH}" = "_$${main_branch}" ]; then \
		while [ "$${secs:=0}" -le ${SLEEP_SEC} ]; do \
			sleep 1; \
			run_id=$$(gh run list --branch "$${main_branch}" --event push --status in_progress --json databaseId --jq '.[0].databaseId'); \
			test -z "$${run_id}" || break; \
			secs=$$((secs + 1)); \
		done; \
		test -n "$${run_id}" || exit 201; \
		gh run watch "$${run_id}" --interval ${SLEEP_SEC} --exit-status; \
		exit; \
	fi; \
	gh pr checks --watch --interval ${SLEEP_SEC} || :

web: .PHONY .SILENT
	gh pr view --web

# gh pr create --upstream/--fork
pr: .PHONY do-pr sleep watch
do-pr: .PHONY
	${GIT_CMD} push --set-upstream origin ${CURRENT_BRANCH}
	gh pr create --fill

merge: .PHONY
	gh pr merge --squash --delete-branch

approve: .PHONY .SILENT
	gh pr review --approve

status: .PHONY .SILENT
	gh pr checks || :
	gh pr status --json latestReviews,reviewDecision --jq '.currentBranch | [.reviewDecision, .latestReviews[0].author.login, .latestReviews[0].state] | join(" ")'

push: .PHONY .SILENT
	${GIT_CMD} push --force --verbose --set-upstream origin ${CURRENT_BRANCH}
