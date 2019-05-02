#!/usr/bin/env bash

set -euo pipefail

if [[ -d $PWD/go-module-cache && ! -d ${GOPATH}/pkg/mod ]]; then
  mkdir -p ${GOPATH}/pkg
  ln -s $PWD/go-module-cache ${GOPATH}/pkg/mod
fi

commit() {
  git commit -a -m "Dependency Upgrade: $1 $2" || true
}

cd "$(dirname "${BASH_SOURCE[0]}")/.."
git config --local user.name "$GIT_USER_NAME"
git config --local user.email ${GIT_USER_EMAIL}

go build -ldflags='-s -w' -o bin/dependency github.com/cloudfoundry/libcfbuildpack/dependency

bin/dependency tomcat "7\.[\d]+\.[\d]+" $(cat ../tomcat-7/version) $(cat ../tomcat-7/uri)  $(cat ../tomcat-7/sha256)
commit tomcat-7 $(cat ../tomcat-7/version)

bin/dependency tomcat "8\.[\d]+\.[\d]+" $(cat ../tomcat-8/version) $(cat ../tomcat-8/uri)  $(cat ../tomcat-8/sha256)
commit tomcat-8 $(cat ../tomcat-8/version)

bin/dependency tomcat "9\.[\d]+\.[\d]+" $(cat ../tomcat-9/version) $(cat ../tomcat-9/uri)  $(cat ../tomcat-9/sha256)
commit tomcat-9 $(cat ../tomcat-9/version)
