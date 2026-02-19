# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.6"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.6/vop-darwin-arm64"
      sha256 "165f409b47a5e2e1d49312064e1198508e3f57b591db50a8965b42e8287215e1"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.6/vop-darwin-amd64"
      sha256 "843d6f5b575e6c3445c4a75b30fcd19d578dd790afe81da8d0ec3f49ec7c5961"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.6/vop-linux-arm64"
      sha256 "16cbcc15ba9c397e882ef620a8a4757dba03b86979b529b8890d23bb1026ab6c"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.6/vop-linux-amd64"
      sha256 "317225281515d1cf420fe92374bc137cf4964935b1c687377513f69472971149"
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
