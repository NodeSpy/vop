# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.10"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.10/vop-darwin-arm64"
      sha256 "e81be560c0afc96fbd53529bff8974d7eb417e877296c493c944070eafa5f08c"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.10/vop-darwin-amd64"
      sha256 "b284b35843f528b3d390559b5ed9f2af10765c89f1a5e83ab2e3c1bc003e589a"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.10/vop-linux-arm64"
      sha256 "0a81d42e5f5c9bc7eba3420eff36908484e9b21afdda349e9cc3d006ae59bbf6"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.10/vop-linux-amd64"
      sha256 "17a58da35638a7535cb2e570a203205f0e65058ad6e645084432852d5d834ef6"
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
