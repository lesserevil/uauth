# uauth

`uauth` is a transparent wrapper that automatically creates reverse SSH tunnels back to your local machine for any ports your command opens — so services running on a remote server are immediately accessible on your laptop without any manual port-forwarding setup.

## How it works

When you SSH into a remote machine and run a command through `uauth`, it:

1. Detects that you are in an SSH session and identifies your client IP.
2. Starts the command as a child process (with full terminal control — raw mode, signals, resize events all work normally).
3. Polls for TCP ports that the child process tree starts listening on.
4. For each new port, establishes a reverse SSH tunnel (`ssh -R`) back to your machine, making the port available on `localhost` on your end.
5. Tears down tunnels automatically when ports stop listening.
6. When the child exits, tears down all tunnels and exits with the same code.

If you are **not** in an SSH session, `uauth` execs the command directly with no overhead.

## Requirements

- Go 1.21+ (to build)
- `ssh` client available on the remote machine
- `sshd` running on your local machine (to accept reverse tunnels)
- Key-based SSH auth from the remote machine back to your local machine (no password/passphrase prompts — tunnels use `BatchMode=yes`)

## Build and install

```bash
# Build only
make build

# Build and install to ~/.local/bin/uauth
make install
```

Make sure `~/.local/bin` is on your `PATH`:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
```

To cross-compile for a specific platform:

```bash
GOOS=linux GOARCH=amd64 go build -o uauth .
```

## SSH setup

### Local machine (your laptop — the SSH client)

Your local machine receives the reverse tunnels, so it must:

**1. Have sshd running and reachable from the remote machine.**

On macOS, enable Remote Login in System Settings > General > Sharing.

On Linux:
```bash
sudo systemctl enable --now sshd
```

**2. Trust the remote machine's SSH key.**

The remote machine will connect back to you as your local user. Add the remote machine's public key to your `~/.ssh/authorized_keys`:

```bash
# On the remote machine, print its public key:
cat ~/.ssh/id_ed25519.pub

# On your local machine, append it:
echo "<remote-public-key>" >> ~/.ssh/authorized_keys
```

If the remote machine has no key pair yet, generate one:
```bash
ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
```

### Remote machine (the server — where uauth runs)

**1. Install `uauth`** — copy the binary to somewhere on `PATH`, e.g. `~/.local/bin/uauth`.

**2. Ensure the remote machine can SSH back to your local machine without prompting.**

Test from the remote machine:
```bash
ssh -o BatchMode=yes <your-local-username>@<your-local-ip> echo ok
```

If this fails, check that the remote's public key is in your local `~/.ssh/authorized_keys`.

**3. (Optional) Pre-trust your local machine's host key** to avoid a first-connection prompt:
```bash
ssh-keyscan <your-local-ip> >> ~/.ssh/known_hosts
```
`uauth` uses `StrictHostKeyChecking=accept-new`, so the first connection auto-trusts the host key — subsequent connections are verified.

## Usage

Prefix any command with `uauth --`:

```bash
uauth -- npm run dev
uauth -- python -m http.server 8080
uauth -- cargo run
```

When run over SSH, any port the command opens on the remote machine becomes available on the same port number on your local `localhost`.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--ssh-user` | `$USER` | Username for the reverse SSH connection back to your local machine |
| `--poll-interval` | `500` | How often to scan for new listening ports, in milliseconds |
| `--verbose` | `false` | Log tunnel lifecycle events to stderr |
| `--log-file` | _(none)_ | Write tunnel events to a file instead of (or in addition to) stderr |

### Example: dev server with verbose logging

```bash
# SSH into the remote machine
ssh myserver

# Run your dev server through uauth
uauth --verbose -- npm run dev
```

Your dev server's port (e.g. 3000) is now reachable at `http://localhost:3000` on your laptop.

### Example: different local username

If your local username differs from your remote username:

```bash
uauth --ssh-user myhomeuser -- python -m http.server 9000
```

## Shell alias / wrapper

To use `uauth` transparently as a drop-in, you can alias commands in your remote `~/.bashrc`:

```bash
alias npm='uauth -- npm'
```

## Notes

- Only ports bound to `localhost` or a wildcard address (0.0.0.0 / `::`) are tunneled. Ports already bound to a specific non-loopback address are ignored.
- All tunnels are torn down cleanly when the child process exits.
- `BROWSER=false` is set in the child process environment to prevent dev servers from trying to open a browser on the remote machine.
