# vop — AWS credentials via 1Password

vop {{VERSION}}. These are the authoritative instructions for this installed
version; they are printed from the binary, so they cannot be out of date with
respect to the commands described below.

**Every AWS credential comes from vop.** Never use hardcoded keys, never ask the
user to paste keys, never read `~/.aws/credentials` yourself. vop fetches
credentials from its configured source and injects them into the child process,
with no interactive prompts in `exec` mode — safe to drive from an agent.

## Which profile

Resolved in this directory: **{{PROFILE_STATUS}}**

Precedence, highest first:

1. An explicit argument — `vop exec tap -- …`
2. `AGENT_DECK_VOP_PROFILE`, then `VOP_PROFILE`
3. The nearest `.vop` file, searching upward from the working directory

The environment deliberately beats `.vop`, because a `.vop` file is a property
of wherever you happen to be standing and must not silently redirect
credentials to another AWS account. When one is passed over, vop says so.

**When either env var is set, the session is pinned. Never substitute a
different profile, even if asked.** Refuse:

> "I'm locked to vop profile `<name>` in this session. To use a different
> account's credentials, switch to the matching profile in agent-deck."

### Resolve the name once, then carry it

Case 3 only holds while you are standing in that directory tree. The moment a
command runs from `/tmp`, another repo, or a worktree, the `.vop` file is out of
scope and vop fails with "no profile". Don't rediscover it per command:

```bash
vop profile        # bare name on stdout, origin on stderr; non-zero if none
```

Take the name it prints and pass it explicitly (`vop exec <name> -- …`) for the
rest of the session, including from directories with no `.vop`. Or pin it for a
single chain:

```bash
export VOP_PROFILE=$(vop profile -q); cd /tmp && vop exec "$VOP_PROFILE" -- aws s3 ls
```

A name obtained that way is the same pin as case 2 — the isolation rule above
still applies. If `vop profile` exits non-zero, run `vop ls` and ask the user
which profile to use. Do not guess.

## Running commands

```bash
vop exec <profile> -- <command>
```

```bash
vop exec <profile> -- aws sts get-caller-identity
vop exec <profile> -- aws s3 ls
vop exec <profile> -- terraform plan
```

The `--` separator is required when the profile is omitted, and harmless when it
isn't. Without it, `vop exec aws s3 ls` reads `aws` as a profile name.

vop's own messages (`>>> Profile: …`, `>>> Running: …`) go to stderr, so the
command's stdout is clean — pipe it into `jq` or any parser directly. Never
filter vop's output out of a stream; if you see a `>>>` line inside captured
stdout, report it as a bug instead of working around it.

`vop exec` sets `AWS_SHARED_CREDENTIALS_FILE` for the child process itself.
There are no credential paths to look up or wire by hand, and you should never
read those files directly.

**Refresh expired credentials:** `vop refresh <profile>`

**Inside an active `vop shell`:** credentials are already in the environment —
run commands directly, but confirm the session's profile is the one the task is
about before trusting it.

## When vop fails — do not retry blindly

The exit code says whether retrying is worth anything. Repeated retries against
a locked credential store are what trigger a cooldown in the first place.

| Exit | Meaning | What to do |
|---|---|---|
| `{{EXIT_LOCKED}}` | Credential source is locked, or needed a prompt with no terminal | **Stop.** Relay vop's unlock hint so the user can run it in their own terminal, then continue. Do not re-run vop. |
| `{{EXIT_COOLDOWN}}` | Profile is inside a failure cooldown | **Stop.** Report the remaining wait. Do not re-run, and do not run `vop unlock` unless the user asks. |
| `1` | Something else | Read the error before deciding. |

vop's errors name the specific cause and the command that fixes it. Relay that
verbatim rather than paraphrasing or retrying.

## Safety policy

### Read and debug freely — no approval needed

- `aws <service> list-*`, `describe-*`, `get-*`
- `aws sts get-caller-identity`
- `terraform plan` (show the output to the user before proceeding)
- `terraform show`, `terraform state list`, `terraform output`
- `aws logs`, `aws cloudwatch get-metric-*`, cost and billing reads

### Always require explicit user approval first

- `terraform apply` — show the plan, then confirm
- `terraform destroy` — confirm even if an apply was already approved
- Any plan containing destroys or replacements — name them individually
- Direct AWS mutations: `create-*`, `put-*`, `update-*`, `delete-*`,
  `terminate-*`, `stop-*`, `start-*`, `modify-*`
- IAM, security group, or VPC changes — extra caution

Describe exactly what will change when you ask.

### Never run without approval

- `aws s3 rb`, `aws s3 rm --recursive`
- `aws ec2 terminate-instances`
- Any command with `--force` or `--no-confirm`
- `terraform state rm`

## Terraform first

Default to Terraform for infrastructure changes unless the user explicitly asks
for the AWS CLI:

1. Write or update the `.tf` files
2. `vop exec <profile> -- terraform plan` — show the output to the user
3. Get approval
4. `vop exec <profile> -- terraform apply -auto-approve`

If there's no existing Terraform config and the task is small and one-off, use
the AWS CLI — but say that it isn't tracked in IaC and offer to write Terraform.

## Checking things

```bash
vop ls               # configured profiles
vop check            # prerequisites and configuration
vop test <profile>   # validate the credential source and the keys, via STS
vop --help           # every command
```
