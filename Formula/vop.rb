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
      sha256 "2be443d09bb27fa7c692b2fabf7d8fc0826dae51b12399a5bc2f3a31b462a48a"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-darwin-amd64"
      sha256 "b1654f6881fed13f7128c94c7dbc04eee150b944fd0533a14107e6c6b9fc71b4"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-linux-arm64"
      sha256 "43edc9f3c7dc3419e838978dbca3309a0b6963ff87a17e61e452af61b043ac16"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.1/vop-linux-amd64"
      sha256 "67f6cf16ef76eeca924050ab5ec79df772cda9d892c1456fdb6b77093b039a6e"
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
