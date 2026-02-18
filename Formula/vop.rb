# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.4"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.4/vop-darwin-arm64"
      sha256 "1fd27e79ba6a0c90d4ec36c9e17ce15edc175e287bcd684c5319840fd4f62258"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.4/vop-darwin-amd64"
      sha256 "3bcd9281e8ccd928cec5181894e1a50e1ed5f1a91918c7f1bac15c271f485191"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.4/vop-linux-arm64"
      sha256 "8eb0eacc0fcee1f9e4d1748125fa679f8453d68f5836b875772b6991e0f7603d"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.4/vop-linux-amd64"
      sha256 "5e8012950fa995222ca8c329e13e0b2caf59bf694dd023f43681adf9277aa028"
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
