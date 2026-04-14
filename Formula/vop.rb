# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.8"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.8/vop-darwin-arm64"
      sha256 "4a774fb285b6d96606968e834e3124260509ee9a13569cfcd16df84579a162c7"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.8/vop-darwin-amd64"
      sha256 "aef289c847cfe42aee93a108df88faa7720ba09effba23178e12c859d23ba66b"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.8/vop-linux-arm64"
      sha256 "29f1b2a8c03c92d588a826e2fe239a1abb5e35cec33312d91eb81179d65b41a7"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.8/vop-linux-amd64"
      sha256 "ea2b5b7fa0b16d7708c2d4d1619f457c98be6b5795d8168cbc935001df12d137"
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
