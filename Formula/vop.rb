# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.3/vop-darwin-arm64"
      sha256 "4e7bc9f9f5e78fd8c5fccefdaec458a73fd6a0e29cf1ba56ec1a8adf5a70964c"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.3/vop-darwin-amd64"
      sha256 "d5a1f17feb8ec42dd359dc4703f390495ce788df7268d93ee570c0f53e0b7215"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.3/vop-linux-arm64"
      sha256 "d67840202b6a912c0ccef385ee3eb2668ee94081316e79e41a9f439eb827b7c0"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.3/vop-linux-amd64"
      sha256 "461a60c5dd2ec9592ae16dfcda9bf48b28295e68311fb613993d442d0c01dcea"
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
