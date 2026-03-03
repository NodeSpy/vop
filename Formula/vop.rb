# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.7"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.7/vop-darwin-arm64"
      sha256 "0b31f29beabcfe06047ebd190e549eafcd062faad1247fbfe18727ac1786175b"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.7/vop-darwin-amd64"
      sha256 "b013ea06792cdc40fa709f0f43fc202426392b862053b70659d3ff5869e32fee"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.7/vop-linux-arm64"
      sha256 "aa683227b9bad9616f2e586afbd6d7d1e910bb5c4f7906722e5974cce40e4410"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.7/vop-linux-amd64"
      sha256 "d8156661b4840b1aff5ff1474444f8232f6f32bd041f87df97189318be6009ba"
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
