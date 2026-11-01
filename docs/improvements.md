# Improvements

Ideas to implement later.

## Testing and CI plan

Today the tests in `internal/server` start a real server on a random port and log in with Go's SSH client. They cover authentication, username rules, connection caps and restarts. They do not cover what users see on screen, how the real `ssh` command behaves, the Docker image, or the deployed server. The plan adds a test layer for each gap, then runs everything in CI.

### Test layers to add

1. **Screen tests for a single user.** Drive the Bubble Tea model with Charm's `teatest` helper at a fixed terminal size and compare the final screen to golden files. Catches layout breakage (e.g. the command bar wrapping) and checks the rate-limit notice. Force colours off and fix the size so golden files do not break for unrelated reasons.
2. **Multi-user chat over SSH.** Two or three SSH clients feed their output into a terminal emulator library to get each screen as text. Assert that alice's screen shows `bob: hi`, that usernames have different colours, and that nothing unexpected is shown. This is the same harness used to record the demo video, with assertions instead of a camera.
3. **The real OpenSSH client.** Go's SSH library and OpenSSH differ in small ways. Run `ssh` under a pseudo-terminal in CI to verify that the rejection explanation (no key, name owned by another key, invalid name) is actually displayed, and that a client offering several keys gets in with the right one.
4. **Container smoke test.** Build the image from a clean checkout, run it with a volume, log in, restart the container, then check that the host key is unchanged and the username is still taken. Also check the healthcheck and that the process runs as non-root. This would have caught the `.gitignore` rule that excluded `internal/identity` from the first commit.
5. **Production check after each deploy.** Log in to `connversed.apps.longitood.com:2222` with a dedicated CI key, check that the lobby appears, and disconnect without sending anything. The probe permanently claims a username such as `ci-probe`, is briefly visible in the lobby, and needs its private key in CI secrets. The same probe could run every few minutes as an uptime check from Grafana.

### CI pipeline (GitHub Actions)

- **Every push and pull request:** formatting check, `go vet`, unit and end-to-end tests, build from a clean checkout, image build and the container smoke test. Layers 2 and 3 run here too once they exist.
- **Gate deploys on CI:** turn off Coolify's auto-deploy for the connversed app and have the CI job trigger the deploy through Coolify's API once everything is green, followed by the production check. Today every push to `main` deploys whether or not tests pass.
- **Optional:** regenerate the demo video in CI and attach it to releases, which doubles as a visual regression check and keeps the website video current.

### Avoiding flaky tests

- Never wait with fixed sleeps; poll for the expected state until a deadline.
- Always use random ports.
- Make randomness controllable: the user colour choice needs a seedable source. The rate limiter already takes a fake clock.
- `go test -race` is currently unreliable: it flags a data race between the SSH library closing the terminal and Bubble Tea's input reader (in `muesli/cancelreader`, which has no newer release) when a session disconnects. It predates the security work and is not fixed by upgrading bubbletea and wish. Run the race detector only on packages that do not start real sessions until upstream fixes it, so a flaky required check does not teach everyone to ignore red builds.
- Sessions that died at startup under CPU load ("error reading input: bad file descriptor", about 2 in 30 full runs) were fixed by upgrading bubbletea to 1.3.10 and wish to 1.4.7. If it ever comes back, treat it as a bug in the stack, not as noise to retry away.

### Suggested order

1. CI workflow with the existing tests and a clean-checkout build.
2. Container smoke test.
3. Gate Coolify deploys on CI.
4. Multi-user screen tests.
5. OpenSSH client test.
6. Production check and uptime monitoring.
