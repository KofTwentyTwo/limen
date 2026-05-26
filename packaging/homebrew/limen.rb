class Limen < Formula
  desc "Terminal launcher TUI for tmux and SSH"
  homepage "https://github.com/KofTwentyTwo/limen"
  version "0.1.0"
  license "GPL-3.0-only"

  depends_on "tmux"

  host_cpu = RbConfig::CONFIG.fetch("host_cpu", "")
  macos_arm = OS.mac? && (Hardware::CPU.arm? || host_cpu == "arm64" || RUBY_PLATFORM.start_with?("arm64-darwin"))
  linux_arm = OS.linux? && (Hardware::CPU.arm? || host_cpu == "aarch64" || host_cpu == "arm64")

  if macos_arm
    url "https://github.com/KofTwentyTwo/limen/releases/download/v#{version}/limen-macos-arm64"
    sha256 "PLACEHOLDER_MACOS_ARM64_SHA256"
  elsif OS.mac?
    url "https://github.com/KofTwentyTwo/limen/releases/download/v#{version}/limen-macos-amd64"
    sha256 "PLACEHOLDER_MACOS_AMD64_SHA256"
  elsif linux_arm
    url "https://github.com/KofTwentyTwo/limen/releases/download/v#{version}/limen-linux-arm64"
    sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"
  else
    url "https://github.com/KofTwentyTwo/limen/releases/download/v#{version}/limen-linux-amd64"
    sha256 "PLACEHOLDER_LINUX_AMD64_SHA256"
  end

  def install
    binary = Dir["limen-*"].first
    chmod 0755, binary
    bin.install binary => "limen"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/limen --version")
  end
end
