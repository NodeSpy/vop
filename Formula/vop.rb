# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.4.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.1/vop-darwin-arm64"
      sha256 "fe1eceb6683885d2b5c9a81cf9f71b0740eab8cd02b3cf26492d2f9cbef09d2e"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.1/vop-darwin-amd64"
      sha256 "9f093c81a8b6eabf19c01759097930d21ef1867b5ceed964e88f95c09366f2a4"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.1/vop-linux-arm64"
      sha256 "2f9fb63afabd1f956917c3087a649547c7708fcbd885670ae1b4f4eca39b3ca5"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.1/vop-linux-amd64"
      sha256 "0beb6e0a24bbd8aac439895a7277db0ec56e289bd1d13e7f15b2c85d6ce9c29b"
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
