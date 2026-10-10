# Release accounts and keys

What Pablo sets up before Quarter's first release (v0.1.0), so the release can publish Homebrew, winget and AUR packages by itself. Each step is from the official documentation, linked at the end of each section and under Sources. Written 2026-10-02 for Batch 8, Phase D (D5 uses the secrets, D7 publishes).

None of this goes into the repo: tokens and keys only ever go into GitHub's secret store (step 4). If one leaks, delete it and make a new one.

| # | What | Needed for | Time | Status |
|---|---|---|---|---|
| 1 | A public repo `stronxyo/homebrew-tap` | `brew install stronxyo/tap/quarter` | 2 min | done 2026-10-10 |
| 2 | A fork of `microsoft/winget-pkgs` | `winget install quarter` | 2 min | to do |
| 3 | One classic GitHub token, scope `public_repo` | steps 1 and 2, from the release | 5 min | to do |
| 4 | Three secrets in `stronxyo/quarter` | the release workflow | 5 min | to do |
| 5 | An AUR account and an SSH key | `yay -S quarter-tui-bin` | 30 min, plus Arch in WSL | to do |
| 6 | GitHub Sponsors (optional) | the README's Sponsors link | 15 min, then GitHub's review | to do |
| | Ko-fi | the `c` key, the README | | done (`ko-fi.com/stronxyo`) |

Steps 1 to 4 are enough for a release with Homebrew and winget. The release publishes to the AUR too once the key from step 5 is in: until the `AUR_KEY` secret exists, the release skips the AUR by itself (`skip_upload` in `.goreleaser.yaml`), so nothing needs switching off.

## 1. The Homebrew tap

A tap is a GitHub repository Homebrew reads extra packages from. Its name must start with `homebrew-`, so `stronxyo/homebrew-tap` is found as `stronxyo/tap`.

1. On github.com, in the upper-right corner, click **+**, then **New repository**.
2. **Owner:** `stronxyo`. **Repository name:** `homebrew-tap`.
3. **Visibility:** **Public** (people's Homebrew must be able to read it).
4. Turn on **Add a README** (this creates the `main` branch the release writes to).
5. Click **Create repository**.

Nothing else: each release adds `Casks/quarter.rb` there by itself.

Sources: [GitHub: Creating a new repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-new-repository), [Homebrew: How to Create and Maintain a Tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap).

## 2. Your fork of winget's package list

winget's packages are files in Microsoft's repository `microsoft/winget-pkgs`. The release writes Quarter's files to your copy of it (a fork) and opens a pull request from there.

1. Open [github.com/microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs).
2. In the top-right corner, click **Fork**.
3. **Owner:** `stronxyo`. Leave the name `winget-pkgs`.
4. Leave the box that copies only the main branch ticked (GitHub's docs call it **Copy the DEFAULT branch only**; for this repository the branch is `master`). It's a very large repository, and the other branches aren't needed.
5. Click **Create fork**.

Sources: [GitHub: Fork a repository](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/fork-a-repo), [GoReleaser: Winget](https://goreleaser.com/customization/publish/winget/).

## 3. One GitHub token

The release needs permission to write to your tap and your fork, and to open a pull request on Microsoft's repository. It must be a **classic** token: GitHub's docs list contributing to public repositories you aren't a member of as something fine-grained tokens can't do.

1. On github.com, click your profile picture (top right), then **Settings**.
2. In the left sidebar, at the bottom, click **Developer settings**.
3. Click **Personal access tokens**, then **Tokens (classic)**.
4. Click **Generate new token**, then **Generate new token (classic)**.
5. **Note:** `quarter releases`.
6. **Expiration:** **Custom**, one year from today, and put a reminder in your calendar: when it expires, make a new one and replace it in step 4. ("No expiration" works too, but GitHub advises against it.)
7. **Scopes:** tick only **public_repo** (under **repo**). Nothing else.
8. Click **Generate token**.
9. **Copy the token now**: GitHub shows it only once. Go straight to step 4 and paste it there. Don't save it in a file or a chat.

Sources: [GitHub: Managing your personal access tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens).

## 4. The secrets in Quarter's repository

The release workflow (D5) reads these by name. Secrets work in private repositories too, so this can be done before Quarter goes public.

1. Open [github.com/stronxyo/quarter](https://github.com/stronxyo/quarter).
2. Click **Settings** (on the repo, not your account).
3. In the sidebar, under **Security**, click **Secrets and variables**, then **Actions**.
4. On the **Secrets** tab, click **New repository secret**, and add each of these, one at a time (**Name**, **Secret**, then **Add secret**):

| Name (exactly) | Secret |
|---|---|
| `HOMEBREW_TAP_TOKEN` | the token from step 3 |
| `WINGET_TOKEN` | the same token from step 3 |
| `AUR_KEY` | after step 5: the whole private key file, from `-----BEGIN OPENSSH PRIVATE KEY-----` to `-----END OPENSSH PRIVATE KEY-----` |

The release itself (making the GitHub release and uploading the downloads) uses GitHub's own automatic token: nothing to add for that.

Sources: [GitHub: Using secrets in GitHub Actions](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-secrets).

## 5. The AUR account and its key

The AUR is where Arch Linux users find community packages. Two things to know first:

- **Registration** reopened on 2026-07-13 with stricter checks: a real (not disposable) email address, which you must confirm through a link within 24 hours.
- **Its captcha asks you to run a command on Arch Linux** (a `pacman -V` line piped through a hash) and paste the answer. Windows can run real Arch Linux through WSL, which has an official Arch image since 2025.

### 5a. Arch Linux in WSL (for the captcha)

1. Open **PowerShell** and run:
   ```
   wsl --install archlinux
   ```
   If WSL isn't on the PC yet (it isn't on yours as of 2026-10-02), Windows installs it first and may ask you to restart; run the line again after restarting.
2. When it finishes, Arch opens in the window. Leave it open for 5b.

Sources: [ArchWiki: Install Arch Linux on WSL](https://wiki.archlinux.org/title/Install_Arch_Linux_on_WSL), [Arch Linux official WSL image](https://antiz.fr/blog/archlinux-official-wsl-image/).

### 5b. Register

1. Open [aur.archlinux.org/register](https://aur.archlinux.org/register).
2. Fill in a username and your email.
3. The captcha shows a command. Copy it exactly as shown, paste it into the Arch window from 5a, press Enter, and paste the short answer it prints into the form.
4. Submit, then open the email from the AUR and confirm within 24 hours.

Sources: [aur-general, 2026-07-13: registration reopened](https://lists.archlinux.org/archives/list/aur-general@lists.archlinux.org/message/TT3OCFFNM6SBMUBKIVTHTKA6UZJNMXIJ/).

### 5c. The SSH key

The release signs in to the AUR with a key instead of a password. It runs unattended, so this key has **no passphrase**, and it's used for nothing else.

1. In **PowerShell** (not Arch), run:
   ```
   ssh-keygen -t ed25519 -f "$env:USERPROFILE\.ssh\aur" -C "quarter AUR release"
   ```
   (`ssh-keygen` comes with Windows; it's in `C:\Windows\System32\OpenSSH`.)
2. When it asks for a passphrase, press **Enter**, and **Enter** again (empty).
3. That makes two files in your `.ssh` folder:
   - `aur.pub`, the **public** key: safe to share.
   - `aur`, the **private** key: treat it like a password.
4. Show the public key, and copy the whole line:
   ```
   Get-Content "$env:USERPROFILE\.ssh\aur.pub"
   ```
5. On the AUR, sign in, click **My Account**, paste that line into **SSH Public Key**, enter your current password at the bottom, and click **Update**.
6. Show the private key, and copy all of it, including the BEGIN and END lines:
   ```
   Get-Content -Raw "$env:USERPROFILE\.ssh\aur"
   ```
   Add it as the secret `AUR_KEY` (step 4).
7. Keep the two files in your `.ssh` folder as a backup, and never put them in a repo.

There's nothing to create on the AUR beyond this: the first release pushes the package `quarter-tui-bin`, which creates it.

Sources: [ArchWiki: AUR submission guidelines, Authentication](https://wiki.archlinux.org/title/AUR_submission_guidelines) (key pair, the public key in My Account), [Microsoft: Key-based authentication in OpenSSH for Windows](https://learn.microsoft.com/en-us/windows-server/administration/openssh/openssh_keymanagement) (`ssh-keygen` on Windows, the two files), [GoReleaser: Arch User Repositories](https://goreleaser.com/customization/publish/aur/).

## 6. GitHub Sponsors (optional)

The README links `github.com/sponsors/stronxyo`. Until this is done, that link shows nothing; say so and Claude takes it out of the README and the Sponsor button.

1. **Two-factor authentication** must be on for your GitHub account first (Settings, then Password and authentication).
2. Open [github.com/sponsors](https://github.com/sponsors) and click **Get sponsored**. Fill in your contact information and choose to be paid to a bank account.
3. Open your Sponsors dashboard (profile picture, **Your sponsors**, **Dashboard**):
   - **Profile:** a short bio and what sponsorship supports.
   - **Sponsor tiers:** your amounts (up to 10 monthly and 10 one-time), then publish them.
   - **Stripe Connect account:** create it; your country of residence and your bank's country must match (Uruguay is supported).
   - **Overview**, then the **tax forms** link: as a non-US resident, the **W-8BEN** form.
4. Click **Request approval**. GitHub reviews it, which "may take a few days".

Sources: [GitHub: Setting up GitHub Sponsors for your personal account](https://docs.github.com/en/sponsors/receiving-sponsorships-through-github-sponsors/setting-up-github-sponsors-for-your-personal-account), [GitHub Sponsors in 30 new regions, Uruguay among them](https://github.blog/news-insights/company-news/github-sponsors-available-in-30-new-regions-2/).

## On release day (D7): what happens, and the one thing to do

- **Homebrew:** the release adds `Casks/quarter.rb` to your tap. `brew install stronxyo/tap/quarter` works right away.
- **AUR:** the release pushes `quarter-tui-bin`. `yay -S quarter-tui-bin` works right away.
- **winget:** the release opens a pull request on `microsoft/winget-pkgs` from your fork. **The first time, a bot asks you to accept Microsoft's Contributor License Agreement: reply on that pull request with exactly**
  ```
  @microsoft-github-policy-service agree
  ```
  (only once, for all Microsoft repositories). Then Microsoft's automatic checks run and the pull request is reviewed; `winget install quarter` works once it's merged. Microsoft gives no fixed time for that. If a bot labels the pull request `Needs-Author-Feedback`, tell Claude what it says.

Sources: [microsoft/winget-pkgs README](https://github.com/microsoft/winget-pkgs/blob/master/README.md) (the CLA), [a winget-pkgs pull request with the CLA reply](https://github.com/microsoft/winget-pkgs/pull/445691).
