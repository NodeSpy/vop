# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.6"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.6/vop-darwin-arm64"
      sha256 "3ad296bcd4adfaae36e55c041a3c067df0f163e4f94db7b58063153b64fd34ad"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.6/vop-darwin-amd64"
      sha256 "c84f3cc089b43d88de3a2b95e89712a727b9f6c59a782fab7e087cd200a5bf41"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.6/vop-linux-arm64"
      sha256 "14d9073bcd20450d9ce17ed15bc9820c7230f333165ceec07ba33f07ffbdc505"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.6/vop-linux-amd64"
      sha256 "55398c277be5ac9f6c1627fa0c5ab449facf35a3b20e63f6ef7cc09c125b442f"
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
