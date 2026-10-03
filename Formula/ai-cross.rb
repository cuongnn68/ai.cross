class AiCross < Formula
  desc "Synchronize AI coding agent instructions with backups and restore"
  homepage "https://github.com/cuongnn68/ai.cross"
  head "https://github.com/cuongnn68/ai.cross.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "."
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/ai-cross version")

    (testpath/"project").mkpath
    (testpath/"instructions.md").write "Use concise answers.\n"
    system bin/"ai-cross", "apply", "--local", "--project", testpath/"project",
           "--file", testpath/"instructions.md"

    assert_equal "Use concise answers.\n", (testpath/"project/AGENTS.md").read
    assert_equal "Use concise answers.\n", (testpath/"project/.github/copilot-instructions.md").read
  end
end
