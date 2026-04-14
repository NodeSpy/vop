# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.9"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.9/vop-darwin-arm64"
      sha256 "2622909f9c84b425fca7eafcaed50ee832460a21463a9d664afeb88ae7d48621"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.9/vop-darwin-amd64"
      sha256 "02010809cb110aa0db94f99599679b666cb3243e2e082e520b9c4f6609cb6acd"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.9/vop-linux-arm64"
      sha256 "983082a3a2ac8c3d123d16fb780b1b4ae4ff3b10b9281e9ccd59a32becd057bc"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.9/vop-linux-amd64"
      sha256 "9cc156cf3c9bacba3515558f9c6a6a58a5bea0a4adc45c251502b76718d76c5c"
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
