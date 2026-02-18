# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.5"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.5/vop-darwin-arm64"
      sha256 "c6961308ec132aec1aac0f0e3acd3b7dfa7a23ea854e764ef8e700da5d9879d1"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.5/vop-darwin-amd64"
      sha256 "5a0f07475bbd52edafef5db76e168ae4d75b6bf0987c8121196be193eafedca9"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.5/vop-linux-arm64"
      sha256 "34981a318f9cc2214eeabbdc74749fa363afd5b2ab64eca20a8262b07eb7fa4a"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.5/vop-linux-amd64"
      sha256 "735801511959617c0e5f3d980a6f705e44944a1d910de29f96c44703a166c9ce"
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
