# typed: false
# frozen_string_literal: true

class Vop < Formula
  desc "AWS credential management via 1Password"
  homepage "https://github.com/NodeSpy/vop"
  version "v0.8.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.8.0/vop-darwin-arm64"
      sha256 "a8edc935e48ad778927d56f3b1ddce87e5ff1326d6a98303c4f3cdf8c9cb2811"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.8.0/vop-darwin-amd64"
      sha256 "b0f455cfe2638b5969cc54311ce3b7481b1e55fe92164e3e7833d862531c2e7e"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NodeSpy/vop/releases/download/v0.8.0/vop-linux-arm64"
      sha256 "c84faa6f1953d9ec2ecd3dfaa28d2302a2ad83b02e951624a7915e09fe9641a1"
    else
      url "https://github.com/NodeSpy/vop/releases/download/v0.8.0/vop-linux-amd64"
      sha256 "91b4b76a3e6c15df03574fa7c3c36ce53e8ff1a628fc62300dc72c81bad56648"
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
