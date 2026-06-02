# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.12"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.12/vop-darwin-arm64"
      sha256 "db4dea30c8e13a5a78ba7c8a5a0c86393b6567040d4d1a46a6126cda23ed13f1"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.12/vop-darwin-amd64"
      sha256 "8f3a136d8fff01ef67ba5f949754d98295654afd9ddab33a90574e258dfd6fde"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.12/vop-linux-arm64"
      sha256 "58377b28679d269c31cfef6816676bf0a2710b5a004e63313f6006c9e0d11ea0"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.12/vop-linux-amd64"
      sha256 "a4667584d15e1afcaa5e0020fd935c793936f27efa34daca0652170fc61e3e24"
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
