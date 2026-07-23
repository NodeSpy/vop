# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.4.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.0/vop-darwin-arm64"
      sha256 "7a45dda224025c8e416ce1cf76d49b7315f8608f1e0b5a80a30aa690637dba07"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.0/vop-darwin-amd64"
      sha256 "d53cd980f7bb9e49822630fb1f64b7a27967664b486798ec47df806abc0f4e98"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.0/vop-linux-arm64"
      sha256 "5adc6ac0469c76b02ce21c49ce110c4f479d3cfd1a49c4b5722a1f19a521be4f"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.4.0/vop-linux-amd64"
      sha256 "3623886cbbb3764967eb23a491c83247e678ec9ed8128618784f7a21b6054bf8"
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
