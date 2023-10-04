#!/bin/bash

set -ueo pipefail

ruby --version
bundle --version

gem update --system

sudo mkdir -p /etc/gemfast
sudo chown -R $USER: /etc/gemfast
cat << CONFIG > /etc/gemfast/gemfast.hcl
caddy {
  port = 80
  host = "http://localhost"
}
license_key = "B7D865-DA12D3-11DA3D-DD81AE-9420D3-V3"
auth "none" {}
cve {
    enabled = true
    max_severity = "medium"
}
CONFIG

if [[ "$BUILD_TYPE" == "docker" ]]; then
  docker load -i gemfast*.tar
  docker run -d --name gemfast -p 80:2020 -v /etc/gemfast:/etc/gemfast -v /var/gemfast:/var/gemfast -v /etc/machine-id:/etc/machine-id gemfast:latest
  sleep 5
  docker ps
  docker logs gemfast
else
    sudo dpkg -i gemfast*.deb
    sudo systemctl start gemfast
    sleep 10
    sudo systemctl status gemfast
    sleep 2
    sudo systemctl status caddy

    journalctl -u gemfast
    journalctl -u caddy
fi

mkdir ./test-cve

pushd test-cve
cat << CONFIG > Gemfile
source "https://rubygems.org"
gem "activerecord", "4.2.0"
CONFIG

bundle config mirror.https://rubygems.org http://localhost
if [[ $(bundle 2>&1 | grep "405") ]]; then
    echo "cve is blocking activerecord 4.2.0"
else
    echo "cve is not blocking activerecord 4.2.0"
    exit 1
fi

popd