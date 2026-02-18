# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.0/vop-darwin-arm64"
      sha256 "a8b759e92f51c6cb9ee48a6c674356e098b386cb514d022b8781e7e77b9a3fe3"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.0/vop-darwin-amd64"
      sha256 "fd95b3fc91e438382bcd94b016622b4c7999b2ee7f78c3958cda06407a03f3ab"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.0/vop-linux-arm64"
      sha256 "dff1a583cb634aca5fe8d19320bbc48161f255dd970d6a368e4caff29162413d"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.0/vop-linux-amd64"
      sha256 "305883dcf9b1301c9835b56aad113a56cd1249b88d8326d4448fc4a9fa5f48ac"
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
