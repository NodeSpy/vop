# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.3.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.0/vop-darwin-arm64"
      sha256 "623eb1a4739c813dc23ba06d46c2e981f443ff5160762cf27a03d7c5f571aa95"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.0/vop-darwin-amd64"
      sha256 "99014b114913f3f91458c35932d119d660d91328a34a27386ca40dec36afbab7"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.0/vop-linux-arm64"
      sha256 "d914e3599ce76148948c1cc97aea5c3bc50235d8777e6b3f0ea24c2e503b144f"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.3.0/vop-linux-amd64"
      sha256 "0f07b7daa3c78bca605099996a93ebcaddf28b92f6d4ee76570ad86ab305765b"
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
