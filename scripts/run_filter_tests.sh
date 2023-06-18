#!/bin/bash

set -ueo pipefail

ruby --version
bundle --version

gem update --system

sudo mkdir -p /etc/gemfast
sudo chown -R $USER: /etc/gemfast
cat << CONFIG > /etc/gemfast/gemfast.hcl
caddy_port = 80
url = "http://localhost"
license_key = "B7D865-DA12D3-11DA3D-DD81AE-9420D3-V3"
auth "none" {}
filter {
    enabled = true
    action = "allow"
    regex = ["stringio"]
}
CONFIG

sudo dpkg -i gemfast*.deb

sudo systemctl start gemfast
sleep 10
sudo systemctl status gemfast
sleep 2
sudo systemctl status caddy

journalctl -u gemfast
journalctl -u caddy

mkdir ./test-filter-allow
pushd test-filter-allow

cat << CONFIG > Gemfile
source "https://rubygems.org"
gem "stringio", "~> 3.0", ">= 3.0.7"
CONFIG

bundle config mirror.https://rubygems.org http://localhost
bundle

popd

cat << CONFIG > /etc/gemfast/gemfast.hcl
caddy_port = 80
url = "http://localhost"
license_key = "B7D865-DA12D3-11DA3D-DD81AE-9420D3-V3"
auth "none" {}
filter {
    enabled = true
    action = "deny"
    regex = ["active.*"]
}
CONFIG

sudo systemctl restart gemfast
sleep 5
sudo systemctl status gemfast
sleep 2
sudo systemctl status caddy

mkdir ./test-filter-deny
pushd test-filter-deny

cat << CONFIG > Gemfile
source "https://rubygems.org"
gem "activesupport", "~> 7.0", ">= 7.0.5"
CONFIG

if [[ $(bundle 2>&1 | grep "405") ]]; then
    echo "Gemfast is working"
else
    echo "Gemfast is not working"
    exit 1
fi

