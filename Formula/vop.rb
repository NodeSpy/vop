# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-darwin-arm64"
      sha256 "3016d8e42094d4897a7135fe01c3bd5a015126f3bfc76eca0631e07814a37c15"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-darwin-amd64"
      sha256 "072bd23f6a5b8b2b6d6dcc0ce471c67afefad1f27bf641c1646a507b56ca588a"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-linux-arm64"
      sha256 "3e9ac2ecfe4d2cd3ab4acea54e965340055fbd59a420bdcc0b9436141e01bc2c"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-linux-amd64"
      sha256 "f05d2d9e18a4bec5ead2ac3d7180abaa590f9f50bfdb035b3af63019e4af6785"
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
