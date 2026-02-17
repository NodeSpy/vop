# typed: false
# frozen_string_literal: true

# Homebrew formula for vop — AWS credential management via 1Password.
#
# Install:
#   brew tap NodeSpy/vop https://github.com/NodeSpy/vop
#   brew install vop
#
# The SHA256 values and version are updated automatically by the
# update-homebrew job in .github/workflows/release.yml.
class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "VERSION_PLACEHOLDER"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/#{version}/vop-darwin-arm64"
      sha256 "SHA256_DARWIN_ARM64_PLACEHOLDER"
    else
      url "https://github.com/NodeSpy/vop/releases/download/#{version}/vop-darwin-amd64"
      sha256 "SHA256_DARWIN_AMD64_PLACEHOLDER"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/#{version}/vop-linux-arm64"
      sha256 "SHA256_LINUX_ARM64_PLACEHOLDER"
    else
      url "https://github.com/NodeSpy/vop/releases/download/#{version}/vop-linux-amd64"
      sha256 "SHA256_LINUX_AMD64_PLACEHOLDER"
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
