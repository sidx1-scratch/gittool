class Gittool < Formula
  desc "A tool that makes Git incredibly simple with 5 core commands"
  homepage "https://github.com/sidx1-scratch/gittool"
  url "https://github.com/sidx1-scratch/gittool/archive/refs/tags/v1.tar.gz"
  sha256 "0355ebd4b11c4e344d534f001fbf998964f9af21ce76400e2a8ed6f32d377ab5"
  license "MIT"

  depends_on "go" => :build

  def install
    # 1. Strict OS Gatekeeping: Block Windows
    if OS.windows? || ENV["OS"] =~ /Windows_NT/i
      odie "Error: gittool does not support Windows operating systems."
    end

    # 2. Compile the Go binary
    system "go", "build", "-ldflags", "-s -w", "-o", bin/"gittool", "main.go"
  end

  test do
    assert_match "gittool - Making Git simple.", shell_output("#{bin}/gittool --help")
  end
end
