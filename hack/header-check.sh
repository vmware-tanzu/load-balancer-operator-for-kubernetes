#!/bin/bash
# Copyright 2020 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

# Simple script that will check files of type .go, .sh, .bash, or Makefile
# for the copyright header.
#
# This will be called by the CI system (with no args) to perform checking and
# fail the job if headers are not correctly set. It can also be called with the
# 'fix' argument to automatically add headers to the missing files.
#
# Check if headers are fine:
#   $ ./hack/header-check.sh
# Check and fix headers:
#   All changes must be committed for fix to work
#   $ ./hack/header-check.sh fix

set -e -o pipefail

# These header variables MUST match the first two lines of the
# VMware-copyright file in the scripts directory.
#
# These will be evaluated as a regex against the target file
HEADER[1]="^\/\/ Copyright [0-9]{4}(-[0-9]{4})? VMware, Inc\.$"
HEADER[2]="^\/\/ SPDX-License-Identifier: Apache-2.0$"

# Initialize vars
ERR=false
FAIL=false

all-files() {
    git ls-files |\
        # Check .go files, Makefile, sh files, bash files, and robot files
        grep -e "\.go$" -e "Makefile$" -e "\.sh$" -e "\.bash$" -e "\.robot$" |\
            # Ignore vendor/
        grep -v vendor/
}

for file in $(all-files); do
  echo -n "Header check: $file... "

  # get the file extension / type
  ext=${file##*.}

  # increment line count in certain cases
  increment=0

  # should we be incrementing the line count
  if [[ $ext == "sh" ]]; then
	  increment=1
  fi

  if [[ "${file#*.}" == "deepcopy.go" ]]; then
	  increment=2
  fi

  for count in $(seq 1 ${#HEADER[@]}); do
    if [[ $ext != "go" ]]; then
        # if not go code assuming # will suffice
        text="${HEADER[$count]/'\/\/'/#}"
      else
        text=${HEADER[$count]}
    fi

    line=$((count + increment ))
    # do we have a header match?
    if [[ ! $(sed "${line}"q\;d "${file}") =~ ${text} ]]; then
      ERR=true
    fi
  done

  if [ $ERR == true ]; then
    # is there is a fix argument and are all changes committed
    if [[ $# -gt 0 && $1 =~ [[:upper:]fix] ]]; then
      # based on file type fix the copyright
      case "$ext" in
        go)
          cat "$(dirname "$0")"/boilerplate.go.txt "${file}" > "${file}".new
          ;;
        sh)
          head -1 "${file}" > "${file}".new
          sed 's/\/\//\#/1' < "$(dirname "$0")"/boilerplate.go.txt >> "${file}".new
          grep -v '#!/bin/bash' "${file}" >> "${file}".new
          ;;
        *)
          sed 's/\/\//\#/1' < "$(dirname "$0")"/boilerplate.go.txt > "${file}".new
          cat "${file}" >> "${file}".new
          ;;
      esac

      if [ "$(uname -s)" = "Darwin" ]; then
        permissions=$(stat -f "%OLp" "${file}")
      else
        permissions=$(stat --format '%a' "${file}")
      fi
      mv "${file}".new "${file}"
      # make permissions the same
      chmod "$permissions" "${file}"
      echo "$(tput -T xterm setaf 3)FIXING$(tput -T xterm sgr0)"
      ERR=false
    else
      echo "$(tput -T xterm setaf 1)FAIL$(tput -T xterm sgr0)"
      ERR=false
      FAIL=true
    fi
  else
    echo "$(tput -T xterm setaf 2)OK$(tput -T xterm sgr0)"
  fi
done

# If we failed one check, return 1
[ $FAIL == true ] && exit 1 || exit 0
