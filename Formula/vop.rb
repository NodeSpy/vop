# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.11"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.11/vop-darwin-arm64"
      sha256 "33f6343f79ce6f74b50c680e601b00263599df39b972722722fb0df07c4cfb31"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.11/vop-darwin-amd64"
      sha256 "b1ce812b51344ba3b82f9c45c073957d78ed9bcc0604166827b2ab256cc50545"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.11/vop-linux-arm64"
      sha256 "0616531d74e72ff7b2dccda4962f2d35b0e18f3ff23c4dd881935c7d1595c431"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.11/vop-linux-amd64"
      sha256 "a7f1790a4efc62db7519f6da0de51fc1415561859cc4c256fe34dc77f89d3e9f"
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
