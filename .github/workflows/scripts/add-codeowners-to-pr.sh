#!/usr/bin/env bash
#
#   Copyright The OpenTelemetry Authors.
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#       http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
#
# Adds code owners without write access as reviewers on a PR. Note that
# the code owners must still be a member of the `open-telemetry`
# organization.
#
# Note that since this script is considered a requirement for PRs,
# it should never fail. 

set -euo pipefail

if [[ -z "${PR:-}" ]]; then
    echo "PR has not been set, please ensure it is set."
    exit 0
fi

main () {
    CUR_DIRECTORY=$(dirname "$0")
    FILES=$(gh pr view "${PR}" --json files | jq -r '.files[].path')
    COMPONENTS=$(grep -oE '^[a-z]+/[a-z/]+ ' < .github/CODEOWNERS)
    REVIEWERS=""
    LABELS=""
    declare -A PROCESSED_COMPONENTS

    for COMPONENT in ${COMPONENTS}; do
        for FILE in ${FILES}; do
            MATCH=$(echo "${FILE}" | grep -E "^${COMPONENT}" || true)
                
            if [[ -z "${MATCH}" ]]; then
                continue
            fi
            
            # If we match a file with a component we don't need to process the file again.
            FILES=$(printf "${FILES}" | grep -v "${FILE}")
            
            if [[ -v PROCESSED_COMPONENTS["${COMPONENT}"] ]]; then
                # We want to remove all files that match a given component, since it is likely
                # that a PR will include multiple changed files for a single component.
                continue
            fi
            
            PROCESSED_COMPONENTS["${COMPONENT}"]=true
        
            OWNERS=$(COMPONENT="${COMPONENT}" bash "${CUR_DIRECTORY}/get-codeowners.sh")

            if [[ -z "${OWNERS}" ]]; then
                # This component has no owners, so there's no point in processing it further.
                break 
             fi

            if [[ -n "${REVIEWERS}" ]]; then
                REVIEWERS+=","
            fi
            REVIEWERS+="$(echo "${OWNERS}" | sed 's/@//g' | sed 's/ /,/g')"
                
            # Convert the CODEOWNERS entry to a label
            COMPONENT_CLEAN=$(echo "${COMPONENT}" | sed -E 's%/$%%')
            TYPE=$(echo "${COMPONENT_CLEAN}" | cut -f1 -d '/' )
            NAME=$(echo "${COMPONENT_CLEAN}" | cut -f2- -d '/' | sed -E "s%${TYPE}\$%%")

            if [[ -n "${LABELS}" ]]; then
                LABELS+=","
            fi
            LABELS+="${TYPE}/${NAME}"
        done
    done
        
    gh pr edit "${PR}" --add-label "${LABELS}" || echo "Failed to add labels to #${PR}"

    # Note that adding the labels above will not trigger any other workflows to
    # add code owners, so we have to do it here.
    gh pr edit "${PR}" --add-reviewer "${REVIEWERS}" || echo "Failed to add reviewers to #${PR}"
}

main || echo "Failed to run $0"
