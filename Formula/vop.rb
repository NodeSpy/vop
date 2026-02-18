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
      sha256 "f345124fbe7af18942927f596c3a16e10fe85ecf84f995a3eddcb719288791cd"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-darwin-amd64"
      sha256 "71a423b96004a927f66fa3197f09bf24e69b40e75edb57003bbd599710471251"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-linux-arm64"
      sha256 "d8d1cca8772f5a192ca6704c82f42352dc73fb802e1717d6ef85578b1c6a7048"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-linux-amd64"
      sha256 "88c109079b5508ecc233e70f8ae42f0152eb475f4a85edce5534b6f197073884"
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
