git fetch origin master

# Exit if the master branch is detected. This isn't a pull request
# and we don't need a CHANGELOG update
if [[ $(git branch | grep 'master\|no branch\|release') ]]; then
    exit 0
fi

if echo $(git diff --name-only $(git rev-parse FETCH_HEAD)) | grep -w CHANGELOG.md > /dev/null; then
    echo "Thanks for making a CHANGELOG update!"
    exit 0
else
    echo "No CHANGELOG update found. Please provide update to CHANGELOG for this change."
    exit 1
fi