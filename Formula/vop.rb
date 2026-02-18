# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.7"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.7/vop-darwin-arm64"
      sha256 "130eae64d00170fded0bbcdc63eb29026fcec6daf918e05ccc432f1d17a4931f"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.7/vop-darwin-amd64"
      sha256 "6f302b3f881a55087159cad750f9837abdcd7e25f1d6382b913b8360f2eaefcf"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.7/vop-linux-arm64"
      sha256 "8d35996cb7947fea02d8504a7ee8f7dbddd87e680fb80871f3ba76b8fa0a4633"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.7/vop-linux-amd64"
      sha256 "73d6eeb4099005d3d43cde76ad84995d685394006210e28c2201d013d5b16ddf"
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
