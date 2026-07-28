# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.6.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.6.0/vop-darwin-arm64"
      sha256 "cff89d5c634563afc1999130dd2070a489c126021b23290445c92685d909f37e"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.6.0/vop-darwin-amd64"
      sha256 "94dd1a8964272810b13e332897c8336ef1af9dc4b9008b4f855258b7377492e7"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.6.0/vop-linux-arm64"
      sha256 "916fabb082255d2dfc440056b71f128cbe57bc2ccc5011b18723477217b8ed03"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.6.0/vop-linux-amd64"
      sha256 "62abd71e74b964da884c5748367e49399aa1e80e67382e2b9f16d8ab68ce72ea"
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
