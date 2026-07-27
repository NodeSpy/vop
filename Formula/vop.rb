# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.5.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.5.0/vop-darwin-arm64"
      sha256 "ee17b054bd2d3698bd40d5fcf8e31c42786728c7d23ec246d4bdd3a817c4b8db"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.5.0/vop-darwin-amd64"
      sha256 "b112e20799124c6517f9497432d32da982806c58a8ca1850cbf5057dd5e5971d"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.5.0/vop-linux-arm64"
      sha256 "332d947695f631dd6332b18c0059b77a69cffc619b62b574366bba1c40ec7607"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.5.0/vop-linux-amd64"
      sha256 "5dca0cee0a305ef1f57ccf57d1734f40298b139c187f6a1799b5db5f8323ed6c"
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
