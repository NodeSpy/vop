# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.3/vop-darwin-arm64"
      sha256 "f1a8c7ea677b0a27d0f190d4de9621328439739eeb19dd4eb8201ddb70b79e56"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.3/vop-darwin-amd64"
      sha256 "2f475d81c9d376038d7fb5627cca67738516d764d55022dce6e46caec7b52cc3"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.3/vop-linux-arm64"
      sha256 "567d57642f9a1dbde16c16c3da5e1a2455207c4014e233cbc008f1dcc970810a"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.3/vop-linux-amd64"
      sha256 "d3c0bd9972833ab54a6f75c29b28377a4301e92cf2c47f3c49e260d630246e17"
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
