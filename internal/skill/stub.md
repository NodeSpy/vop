---
name: vop
description: "AWS credentials via 1Password (vop). Use whenever AWS credentials are needed — AWS CLI, Terraform, or any AWS SDK usage. Covers credential injection via vop, profile isolation enforcement, read-only safety policy, and destructive-action approval gates."
---

<!-- vop skill stub v{{STUB_VERSION}} — installed by `vop skill install`.
     Deliberately short: the full instructions live in the vop binary so they
     can never drift from the installed version. Run `vop skill` to read them. -->

# vop — AWS credentials via 1Password

**Every AWS credential comes from vop.** Never use hardcoded keys, never ask the
user to paste keys, never read `~/.aws/credentials` yourself. vop is the only
credential source in this environment.

## Before your first AWS command in a session, run this

```bash
vop skill
```

It prints the full instructions for the installed version — profile resolution
and isolation rules, the safety policy, approval gates, exit-code handling — and
names the profile that resolves in the current directory. Follow what it prints;
where it differs from this file, it wins.

If `vop` is not installed, stop and tell the user. Do not fall back to any other
credential source.

## The minimum, if you read nothing else

- Run AWS work through vop: `vop exec <profile> -- <command>`. The `--` is
  required; without it `vop exec aws s3 ls` reads `aws` as a profile name.
- `vop profile` prints the profile that resolves here — the bare name on stdout.
  Resolve it once and reuse it; a `.vop` file is out of scope the moment you run
  from another directory.
- vop's own `>>>` messages go to stderr, so a command's stdout stays clean for
  `jq` and other parsers. Don't filter vop's output.
- **Exit 11** (failure cooldown) or **Exit 12** (credential source locked):
  stop, report vop's message verbatim, and do not retry. Retrying is what
  extends the cooldown.
- Reads (`describe-*`, `list-*`, `get-*`, `terraform plan`) need no approval.
  Anything that creates, modifies, or deletes an AWS resource needs explicit
  user approval first — describe exactly what will change when you ask.
