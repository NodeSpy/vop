# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.1/vop-darwin-arm64"
      sha256 "38c7c517f2dd0de1b422320b505664894ee5dd8ab988eb32eb0fdcf127d8b1f6"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.1/vop-darwin-amd64"
      sha256 "a478062a16733780141ca30149aaf8fddbfdf07002131d5a06ed53e6245da254"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.1/vop-linux-arm64"
      sha256 "46f78a34e7e8036337f2cd29d51f4e651624d7933c21bf658840bd87a8a8e550"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.1/vop-linux-amd64"
      sha256 "ce35783b6f11f9879685a0469eb5f84455c0bfe25a8db16c5ab57e034110c914"
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
