class Gittool < Formula
  desc "A tool that makes Git incredibly simple with 5 core commands"
  homepage "https://github.com/sidx1-scratch/gittool"
  url "https://github.com/sidx1-scratch/gittool/archive/refs/tags/v4.tar.gz"
  sha256 "6eb05e5017f3abb8df54d6f2b51c9761ae14635603f3170ca3ff59bb88b2c82e"
  license "MIT"

  depends_on "go" => :build

  def install
    # 1. Strict OS Gatekeeping: Block Windows
    if ENV["OS"] =~ /Windows_NT/i
      odie "Error: gittool does not support Windows operating systems."
    end

    # 2. Compile the Go binary
    system "go", "build", "-ldflags", "-s -w", "-o", bin/"gittool", "main.go"
  end

  test do
    assert_match "gittool - Making Git simple.", shell_output("#{bin}/gittool --help")
  end
end
