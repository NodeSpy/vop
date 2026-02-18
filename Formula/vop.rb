# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.4"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.4/vop-darwin-arm64"
      sha256 "d162188eed07d8ff028f14aa4d2291931503e2ed12861965ec4e4980aac437ae"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.4/vop-darwin-amd64"
      sha256 "0ae075636ec7b2f6e63f1dea467d67f093229954d73fb357e8afcb0b238bf93d"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.4/vop-linux-arm64"
      sha256 "64973f86388a509c4b96b2b4ce301c103717ce9eef43ec72ed05980980a937a3"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.4/vop-linux-amd64"
      sha256 "613a928e328d76de43ac3498d04a2177615760ee72d9fe9b6bb6cfcd88d913f2"
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
