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
  osarchs=(
    # "aix ppc64"
    # "android amd64"
    # "android arm"
    # "android arm64"
    "darwin amd64"
    "darwin arm64"
    # "dragonfly amd64"
    # "freebsd 386"
    # "freebsd amd64"
    # "freebsd arm"
    # "illumos amd64"
    # "ios arm64"
    "js wasm"
    "linux 386"
    "linux amd64"
    "linux arm"
    "linux arm64"
    "linux ppc64"
    "linux ppc64le"
    "linux mips"
    "linux mipsle"
    "linux mips64"
    "linux mips64le"
    "linux riscv64"
    "linux s390x"
    # "netbsd 386"
    # "netbsd amd64"
    # "netbsd arm"
    # "openbsd 386"
    # "openbsd amd64"
    # "openbsd arm"
    # "openbsd arm64"
    # "plan9 386"
    # "plan9 amd64"
    # "plan9 arm"
    # "solaris amd64"
    "windows 386"
    "windows amd64"
    "windows arm"
    "windows arm64"
  )
  set +e
  for i in "${osarchs[@]}"; do
    osarch=($i)
    os=${osarch[0]}
    arch=${osarch[1]}
    suffix=
    [[ $os = "windows" ]] && suffix=".exe"
    [[ $arch = "wasm" ]] && suffix=".wasm"
    out="${outdir}/${app}_${os}_$arch$suffix"
    printf "building \"$out\" ... "
    CGO_ENABLED=1 GOOS=$os GOARCH=$arch \
      go build -trimpath -ldflags="$ldflags" -o $out $pkg \
      && echo "SUCCESS" \
      || echo "FAILED"
  done
  set -e
else
  go run -trimpath -ldflags="$ldflags" $pkg $@
fi