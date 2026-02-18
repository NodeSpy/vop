# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.2"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.2/vop-darwin-arm64"
      sha256 "aeca12243e20f3fbc760860ce0600eb867ba9d695d9c22295cb2a0dbbe6fa48b"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.2/vop-darwin-amd64"
      sha256 "1d723ce58104defbcea24ff4cb08c6c9fb81f1e51a1c5f2846fc925142e75deb"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.2/vop-linux-arm64"
      sha256 "5c4ad15c626dae67190d3a94c465ffc080efecf76b2458b33698b14e7efdb3c0"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.2/vop-linux-amd64"
      sha256 "6dee7ce7ace7cdf40260271db3876cd60d178f75f566d95241791041723becd5"
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
