# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.8"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.8/vop-darwin-arm64"
      sha256 "2c0325622cce78101a27c7aee4df8dd3d7e9f7b613b322c2b2758d3d52fea530"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.8/vop-darwin-amd64"
      sha256 "78287a35c57b8696b4a9eb2ad5da7f9fffc1c354408a613cdb33819392688424"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.8/vop-linux-arm64"
      sha256 "50a4712e9cf1f82634956a43723566ede2241f9eff78b485d5b1a3f85d41e380"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.8/vop-linux-amd64"
      sha256 "f0c4da49217a5c65474e88e8d3f35d2859011f0ade693680b205b40a2026e7b1"
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
