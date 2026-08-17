# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.8.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.8.1/vop-darwin-arm64"
      sha256 "40450fb3f121060269118569e4a61f7c8271bb227ae43fefc0642bd26e8b4bb6"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.8.1/vop-darwin-amd64"
      sha256 "fcd9d71cddbee771ccbaa522fbc83d2a1aba29602b58422905504e276a77d701"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.8.1/vop-linux-arm64"
      sha256 "6d50b795181dbd959e6c40de0ad476c8459b7da9fb8b13e1332d941183491e5d"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.8.1/vop-linux-amd64"
      sha256 "732bf940a0f3a02a53d4b588644731898778ff3fc01997bf855521c2379ae39b"
    end
  end

  def install
    binary = Dir["vop-*"].first || "vop"
    bin.install binary => "vop"
  end

  def caveats
    <<~EOS
      To let AI agents use vop for AWS credentials, install the vop skill:
        vop skill install

      It writes a short pointer to ~/.claude/skills/vop/SKILL.md. The full
      instructions are printed by 'vop skill' straight from the binary, so
      they track upgrades and the stub never needs reinstalling.
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/vop version")
  end
end
