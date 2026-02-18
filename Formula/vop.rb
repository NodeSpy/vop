# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.5"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.5/vop-darwin-arm64"
      sha256 "b6a25e23bc065acba311bcf907c3ffb2edef5042270be2b1f6769b30fb3b0d44"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.5/vop-darwin-amd64"
      sha256 "bd0f85eb020f8f0f8785773d2acb057592fb4fd5fa2712c93b179dcec5fc4d3b"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.5/vop-linux-arm64"
      sha256 "4d03d7c229d99976a123f311668f9e33a40bd93fbff1b4fbf8ee77c99b17891f"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.5/vop-linux-amd64"
      sha256 "717898e4c8ec3e1b123dc0531e5743255d319709532fc67f1a2265865f37a1e5"
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
