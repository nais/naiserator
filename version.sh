#!/bin/sh
dirty=""
test -z "$(git ls-files --exclude-standard --others)"
if [ $? -ne 0 ]; then
  dirty="-dirty"
fi
echo $(date "+%Y-%m-%d")-$(git --no-pager log -1 --pretty=%h)${dirty}
