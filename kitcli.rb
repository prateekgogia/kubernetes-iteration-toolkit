# Uupdate the version and SHA256 for the CLI
require 'formula'
class Kitcli < Formula
  homepage 'https://github.com/awslabs/kubernetes-iteration-toolkit/substrate'
  version 'v0.0.9'
  if Hardware::CPU.is_64_bit?
    url 'https://github.com/prateekgogia/kubernetes-iteration-toolkit/releases/download/v0.0.9/kitcli_v0.0.9_darwin_amd64.zip'
    sha256 '228e9423813950beb149b8890f8bb9911424a1dd49664686948b340b50dc3a22'
  else
    echo "Hardware not supported"
    exit 1
  end
  def install
    bin.install 'kitcli'
  end
end