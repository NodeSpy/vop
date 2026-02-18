# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.2.6"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.6/vop-darwin-arm64"
      sha256 "833321a7428a0349f58c80ae31b3844b37a97fbdf3923e391623238867bec02c"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.6/vop-darwin-amd64"
      sha256 "3e0f3bfb40f3bb8b758956f5b0efc29c192c08e9c082fa4651177b9968927aaa"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.6/vop-linux-arm64"
      sha256 "56ce9d0445a863ee3f85e9c89e55383719305b06a97312b1f695fa79fa391711"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.2.6/vop-linux-amd64"
      sha256 "757d828e0e0de54e9e9432dc9e83663480a596eda3577543e5ac790c3b9c94be"
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
