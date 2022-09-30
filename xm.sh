#!/bin/bash

set -e

pkg=github.com/jamesliu96/xigma/cmd/xm
app=xm
tag=$(git describe --tags --always)
rev=$(git rev-list -1 HEAD)
ldflags="-X main.gitTag=$tag -X main.gitRev=$rev"
outdir=build
echo "# $pkg $tag $rev" 1>&2

if [[ $1 = "-build" ]]; then
  printf "removing \"$outdir\" ... "
  rm -rf $outdir && echo "SUCCESS" || echo "FAILED"
  ldflags="$ldflags -s -w"
  osarchs=$(go tool dist list)
  set +e
  for i in $osarchs; do
    IFS="/"
    osarch=($i)
    unset IFS
    os=${osarch[0]}
    arch=${osarch[1]}
    suffix=
    [[ $os = "android" || $os = "ios" || $os = "js" ]] && continue
    [[ $os = "windows" ]] && suffix=".exe"
    [[ $arch = "wasm" ]] && suffix=".wasm"
    out="${outdir}/${app}_${os}_$arch$suffix"
    printf "building \"$out\" ... "
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch \
      go build -trimpath -ldflags="$ldflags" -o $out $pkg \
      && echo "SUCCEEDED" \
      || echo "FAILED"
  done
  set -e
else
  go run -trimpath -ldflags="$ldflags" $pkg $@
fi