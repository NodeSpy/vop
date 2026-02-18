# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-darwin-arm64"
      sha256 "479b140d02115639dbc778e7e6e46994af5f058a5cccf8c48f01226b90f7f0aa"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-darwin-amd64"
      sha256 "6c26d0bed22d3110abbfb1455f97398dee565210d856f50ed62e79a875dddc11"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-linux-arm64"
      sha256 "21dd7bc638efc0177499285eea25924f5a24836d7af5bca5d8336ca63ab75b18"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-linux-amd64"
      sha256 "68e413261703b2d901c717588c6430ef2631ad627155af082ce4797caaa0c4d8"
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
