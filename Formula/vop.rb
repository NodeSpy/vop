# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.1.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.3/vop-darwin-arm64"
      sha256 "0c93755082d6bd4a86442094859bd4f5700c4119ad358892f1bcab787c9f7c54"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.3/vop-darwin-amd64"
      sha256 "522fdcaa4407ca28670b2743471118c1804d92fa687b8783be45c3c6cbd19a91"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.3/vop-linux-arm64"
      sha256 "283548e108d646e52348b97c24581e1e77789a7afc6bb75e874ad7bd2f4bd027"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.1.3/vop-linux-amd64"
      sha256 "68f147b5072078bd0049a58b18cd052bd8935ac03ef788242fcd8520642b2cb6"
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
