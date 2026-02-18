# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.4"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.4/vop-darwin-arm64"
      sha256 "c3fd562f0c04b67a39eca9c06bc4eb667a1de386785378f1f776ba34e27bd668"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.4/vop-darwin-amd64"
      sha256 "b681f182fb8bad9fddfddf9b1332ddaa78b4561f23cc5cb462a137bdc019e5d7"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.4/vop-linux-arm64"
      sha256 "ef5217013a6d9925923071bed0451285ac47350b3dda259f0873beee300d37eb"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.4/vop-linux-amd64"
      sha256 "92a3963312beb033ea469850e65d0615c87d0de4e0ae6453fe9ba0a6def137e9"
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
