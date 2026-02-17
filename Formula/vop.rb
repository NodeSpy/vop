# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.0/vop-darwin-arm64"
      sha256 "9afa345c2123e2e40469b9b80f818d62763a17c3c6db1b2dab77ae7917ff3f40"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.0/vop-darwin-amd64"
      sha256 "6783197dbc1c2d73f4bbebfc6b93263594c707951591de9a3a42479d936a6039"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.0/vop-linux-arm64"
      sha256 "64d2d8f3adb0d4e1198d00bca4425f1f519332aa5796fc0ec26e23d4440a06fe"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.0/vop-linux-amd64"
      sha256 "e872ecf9274bc6534dd3f8529b48cf2734b05b1f8f11a0687d577662295bc049"
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
