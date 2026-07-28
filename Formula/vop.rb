# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.7.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.7.0/vop-darwin-arm64"
      sha256 "f1c7823cd5f82138da911f2caebfa13fa32f798b37b325bb1a4174b2db74ebe3"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.7.0/vop-darwin-amd64"
      sha256 "3f5e65edc83115d6514063898059fc43741b9487b9b8a60e5ff15841a7403730"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.7.0/vop-linux-arm64"
      sha256 "89094eb7677a3263aa96aab18ffe7602ea3edd0a75aaf219a090ee85b361256b"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.7.0/vop-linux-amd64"
      sha256 "13c22831e5779aea9dbead652dc7fdf9aa4a87816ed87dc604fd3aadb1f52939"
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
