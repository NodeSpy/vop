# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.2/vop-darwin-arm64"
      sha256 "6f7e7c9b298f85761d48687fe59b2e28ec4de0f790d0abeadaba78c903d10d78"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.2/vop-darwin-amd64"
      sha256 "27092edd140664ce36f068f3886d69800e34d5d1fe82eba71c3e006000b427dc"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.2/vop-linux-arm64"
      sha256 "2c4e850d10aa0f49408b49440ca65ec78ef52f8bbdce8c40fefcefc24dd5f7ae"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.2/vop-linux-amd64"
      sha256 "592b57b5a2dbb07b28d569278a6b3fa4533475a93681b3dff74375a5038f66d0"
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
