# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.4.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.2/vop-darwin-arm64"
      sha256 "b3efa11be052d28ebe2ba4ed0b9697e1c9ca39d9cc84d4513f371afcfe341e12"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.2/vop-darwin-amd64"
      sha256 "8485c243e88f624547a2991a13ef7b1fc0bce07f6ae42813f756111a1d8d7a9c"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.2/vop-linux-arm64"
      sha256 "80c605e7be4f725ce6e14d0f4caddb259228aa138a6edb6f96433029187ff77d"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.2/vop-linux-amd64"
      sha256 "0fc83fe633574accb420678255a084e90e2c7b1b5d0ecc88bf363eed5a511d2b"
    end
  end

  def install
    binary = Dir["vop-*"].first || "vop"
    bin.install binary => "vop"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/vop version")
  end
end
