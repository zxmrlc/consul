# This script verifies whether all merged PRs with a backport/* label are cherry picked to the $GIT_BRANCH (ex: release/1.7.x)
# Parameters:
#    GIT_BRANCH: the release branch to check backports for

#!/bin/bash
# no set -e because we want to print out all the non-merged PRs at once rather than fail on the first one
set -uo pipefail

# Set variables
GIT_BRANCH=${GIT_BRANCH:=$(git rev-parse --abbrev-ref HEAD)}
REPONAME=$(basename `git rev-parse --show-toplevel`)

# check if we are looking at a valid git branch to check; needs to be release/#.#.x
if ! [[ ${GIT_BRANCH} =~ ^release\/[0-9]+\.[0-9]+\.x ]]; then
    echo "Cannot check this branch ($GIT_BRANCH); it must be a release branch, ex: release/1.7.x"
    exit 1
fi

# Fetch all git tags
git fetch
git fetch --tags

# Parse out the x.y version to check the latest tag
minor_version=$(echo ${GIT_BRANCH} | awk -F '[/.]' '{print "v"$2"."$3}')
# Get the latest tag with that minor version, grep -v any prereleases
latest_minor_tag=$(git tag -l | grep ${minor_version} | grep -v "-" |sort -rV | head -n1)
# Get date of that latest minor version tag
latest_minor_date=$(git log -1 --format=%ad --date=short  ${latest_minor_tag})

# get all merged PRs since that latest minor tag date that has a backport label for the release branch; ex: branch: release/1.7.x -> label: backport/1.7
pr_numbers=$(curl -s "https://api.github.com/search/issues?q=repo:hashicorp/${REPONAME}+is:pr+is:merged+merged:%3E${latest_minor_date}+label:backport/${minor_version#v}" | jq '.items[] | .number')

# check that merge_commit_sha of all PRs merged in since the last minor release is now in the release branch
for num in $pr_numbers; do
    merged_commit=($(curl -s "https://api.github.com/repos/hashicorp/${REPONAME}/pulls/${num}" | jq --raw-output '.merge_commit_sha'))
    git merge-base --is-ancestor  ${merged_commit} $(git rev-parse ${GIT_BRANCH})
    if [ $? -ne 0 ]; then
        echo "https://github.com/hashicorp/${REPONAME}/pull/${num} is not cherry-picked into ${GIT_BRANCH}"
    fi
done