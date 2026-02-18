# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.5"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.5/vop-darwin-arm64"
      sha256 "107e3c8d2b5f709ce7505b8218dd83fc216410ce836a7513614f9e80b52f9aab"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.5/vop-darwin-amd64"
      sha256 "f50acc9b14cb55a59e15ceb82130cc323e4edd73299d14965e7c00a88789e6b3"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.5/vop-linux-arm64"
      sha256 "ee1be7bdb2b8ad6fb7b98146cdaa52b0e7b284a1a1ec02caf62a148134e5609c"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.5/vop-linux-amd64"
      sha256 "c9337374d08b12db330f6bbee07274ecb5458847fcd0a19926470719892cdba8"
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
