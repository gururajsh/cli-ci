#!/usr/bin/env bash

set -eu

DEBIAN_FRONTEND=noninteractive
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
CF6_TEMPLATE="$DIR/../ci/installers/deb/control_v6.template"

log() {
  echo "==> $1"
}

create_installer() {
  arch="$1"
  template_dir="$2"
  version="$3"

  mkdir -p "$template_dir/DEBIAN"
  template="$template_dir/DEBIAN/control"
  log "$version: Creating $arch installer at $template..."

  # SIZE="$(BLOCKSIZE=1000 du $root/extracted-binaries/cf-cli_linux_x86-64 | cut -f 1)"
  cp ${CF6_TEMPLATE} $template
  # echo "Installed-Size: ${SIZE}" >> $template 
  echo "Version: $version" >> $template
  echo "Architecture: amd64" >> $template
  echo "Depends: cf-cli (= $version)" >> $template


  fakeroot dpkg --build $template_dir $dir/cf-cli-installer_x86-64.deb
  # TODO: sign_installer $dir/$arch/*
}

create_installers_for_version_dir() {
  dir=$1
  version="$(echo $1 | tr -d v)"
  local templates_dir="$2"
  installer_template="$templates_dir/cf6_$version"

  for file in $dir/*.deb; do
    if [[ ! -f $file ]]; then
      log "$version: Skipping because it had no existing deb files..."
      return
    fi
    break
  done

  # create_installer "i386" "$installer_template" "$version"
  create_installer "x86_64" "$installer_template" "$version"
}

sign_installer() {
  installer_path="$1"

  # workaround for phusion base image; which prompts for a pass phrase
  cat<<EOF >sign-rpm
#!/usr/bin/expect -f
spawn rpmsign --addsign {*}\$argv
expect -exact "Enter pass phrase: "
send -- "\r"
expect eof
EOF
  chmod 700 sign-rpm

  ./sign-rpm $installer_path

  # log "Signing $installer_path..."
  # rpmsign --addsign $installer_path

  rm -f sign-rpm
}

init_gpg() {
  gpg_dir=$1
  gpg_key_location=$2

  export GNUPGHOME=$gpg_dir
  chmod 700 $GNUPGHOME

  gpg --import $gpg_key_location

  cat >> $GNUPGHOME/gpg.conf <<EOF
personal-digest-preferences SHA256
cert-digest-algo SHA256
default-preference-list SHA512 SHA384 SHA256 SHA224 AES256 AES192 AES CAST5 ZLIB BZIP2 ZIP Uncompressed
EOF
}

main() {
  # Test usage: create_virtual_debian_v6_installers.sh "s3://cf-cli-dev/releases" "/tmp/gpg_dir/privatekey.asc"
  # Prod usage:
  # > sudo docker run -v ~/workspace/cli-ci:/tmp/cli-ci -v /Volumes/certificates/:/tmp/certificates -it cfcli/cli-package /bin/bash
  # > cd /tmp/cli-ci
  # > aws configure # enter the Administrator credentials
  # > bin/create_virtual_debian_v6_installers.sh "s3://cf-cli-releases/releases/" "/tmp/certificates/Cloudfoundry-Private-PGP.gpg"
  # > aws s3 sync $release_dir s3://cf-cli-releases/releases # should upload all cf6 rpms
  # > ?????createrepo -s sha $release_dir # creates a repodata folder
  # > aws sync $release_dir/repodata s3://cf-cli-rpm-repo/repodata

  s3_bucket_path=$1
  gpg_key_location=$2
  # release_dir="$(mktemp -d)"
  release_dir="/tmp/tmp.0Moqt0vJkI"
  templates_dir="$(mktemp -d)"
  gpg_dir="$(mktemp -d)"

  trap "rm -rf $gpg_dir" 0

  log "Release path: $release_dir"
  log "Template path: $templates_dir"
  log "GPG path: $gpg_dir"

  # aws s3 sync $s3_bucket_path $release_dir


  cd $release_dir

  init_gpg $gpg_dir $gpg_key_location

  for d in *v6* ; do
    create_installers_for_version_dir $d $templates_dir
    echo
  done
}

main "$@"
