# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.1.4"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.4/vop-darwin-arm64"
      sha256 "6e5e95298f9b743c011969799d68f9cd741d3b713abaef35d795d7ebb63e6e04"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.4/vop-darwin-amd64"
      sha256 "36b415509eab9aa9c30583fafb2eaf7208632f8036e54e3746c351c874514827"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.4/vop-linux-arm64"
      sha256 "12a73ac5d1370f2509e01b0323eae243f65f340dc64ecd2b7c356649155d41d1"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.4/vop-linux-amd64"
      sha256 "311595f4628c8d951c8bdf105b4e262d033af34c6368ff2fbc927825b4bdf06e"
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
