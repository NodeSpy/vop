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
      sha256 "b09237dbab7379c39bf3d0801263b9b8244fa1385076726ede775d3676eb79f4"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.3/vop-darwin-amd64"
      sha256 "36f6ef80f4acd86db674e34f12a12aa2460acf53cb8aa37592c03a48d4183ecb"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.3/vop-linux-arm64"
      sha256 "53a28009e207f352ed9bc55f3b7a1ff79a2ab0bcab6aa553a28b35ed56b33e98"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.3/vop-linux-amd64"
      sha256 "c1be259606ba53bdee4157625201dee46ed040e90a35ea8186d89c6aa558edd2"
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
