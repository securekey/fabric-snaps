#!/bin/bash
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

CHECK=$(git diff --name-only HEAD --diff-filter=ACMRTUXB * | grep -v .png$ | grep -v .rst$ | grep -v .git \
  | grep -v .pem$ | grep -v .block$ | grep -v .tx$ | grep -v ^LICENSE$ | grep -v _sk$ \
  | grep -v .key$ | grep -v \\.gen.go$ \
  | grep -v .md$ | grep -v ^vendor/ | grep -v ^build/ | grep -v .pb.go$ | sort -u)

if [[ -z "$CHECK" ]]; then
  CHECK=$(git diff-tree --no-commit-id --name-only --diff-filter=ACMRTUXB -r $(git log -2 \
    --pretty=format:"%h") | grep -v .png$ | grep -v .rst$ | grep -v .git \
    | grep -v .pem$ | grep -v .block$ | grep -v .tx$ | grep -v ^LICENSE$ | grep -v _sk$ \
    | grep -v .key$ | grep -v \\.gen.go$ \
    | grep -v .md$ | grep -v ^vendor/ | grep -v ^build/ | grep -v .pb.go$ | sort -u)
fi

echo "Checking committed files for SPDX-License-Identifier headers ..."
missing=`echo $CHECK | xargs grep -L "SPDX-License-Identifier"`
if [ -z "$missing" ]; then
   echo "All files have SPDX-License-Identifier headers"
   exit 0
fi
echo "The following files are missing SPDX-License-Identifier headers:"
echo "$missing"
echo
echo "Please replace the Apache license header comment text with:"
echo "SPDX-License-Identifier: Apache-2.0"
exit 1
