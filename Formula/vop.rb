# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.1.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.1/vop-darwin-arm64"
      sha256 "4ceab68c17782a78d5bb5d9b284e03df81fe46aeced081af1813fd70626b5683"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.1/vop-darwin-amd64"
      sha256 "cea6d3c8e2b4c61a2c47a763962fd34b5d696eda8100f294f3cdef0a069833e1"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.1/vop-linux-arm64"
      sha256 "bd91298ce2b56245196d91fa575c266aaef16889e8bc525e3a7b2a691b4c7fc6"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.1/vop-linux-amd64"
      sha256 "22df2c5862427624ddcc3015b19ecfb5519b2e4cc0a1a79f9ecbed16594a1bab"
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
