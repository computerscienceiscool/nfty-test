# nfty-test

A Go CLI that sends push notifications to your team's phones using [ntfy](https://ntfy.sh). Pipe in commit logs, alerts, or any text — your team gets notified instantly.

## Why this matters

Every push notification that arrives on your phone goes through a single gatekeeper: **Google (FCM)** on Android and **Apple (APNs)** on iPhone. There is no way around this. No matter what app you use — Slack, Teams, email, or ntfy — the actual delivery of that push notification is controlled by Google or Apple. They own the pipe between the internet and your lock screen.

This means if you want to build any kind of real-time team alerting — deploy notifications, CI failures, monitoring alerts — you cannot just open a socket to someone's phone and send a message. You must go through their infrastructure, which means you need an app registered with their push notification services, signed with their certificates, and distributed through their app stores.

Building your own push notification system from scratch would require registering as a developer with both Apple and Google, maintaining an app in both stores, managing push certificates and tokens, and keeping up with their evolving requirements. For most teams, that is not a reasonable investment for what amounts to "tell someone something happened."

**What about SMS?** Sending text messages programmatically is technically possible through services like Twilio, but it comes with its own wall of complexity. In the US, carriers now require **10DLC registration** (Ten-Digit Long Code) for any application-to-person messaging. This involves brand verification, campaign registration, and carrier-by-carrier approval — a process that can take days to weeks. Unregistered messages are silently filtered and never delivered. You would still be paying a wholesale SMS aggregator somewhere in the chain, and you would be subject to per-message costs and rate limits imposed by AT&T, T-Mobile, and Verizon individually. For team notifications, SMS is solving the wrong problem at the wrong price.

**ntfy sidesteps all of this.** It is an open-source push notification service with a free public server. It handles the Google and Apple integration for you. There is no API key, no account required, no app store approval process on your end, and no per-message cost. You publish a message to a topic over HTTP, and anyone subscribed to that topic gets a push notification. That's it.

## Sign up

There is no sign-up. The free tier on `ntfy.sh` supports:

- **250 messages per day** (per IP)
- **2 MB attachments**
- **No account required**

You only need an account if you want reserved topic names or access control. For team use, just pick a unique topic name that's hard to guess — the topic name acts as a shared secret.

## Receiving notifications on your phone

### Install the app

- **Android**: [Google Play Store](https://play.google.com/store/apps/details?id=io.heckel.ntfy)
- **iPhone**: [Apple App Store](https://apps.apple.com/app/ntfy/id1625396347)

### Subscribe to a topic

1. Open the ntfy app
2. Tap the **+** button
3. Enter your team's topic name (e.g. `myteam-alerts-2026`)
4. Make sure the server is set to `https://ntfy.sh`
5. Tap **Subscribe**

You will now receive push notifications for every message published to that topic.

### Browser alternative

If you don't want to install the app, open `https://ntfy.sh/your-topic-name` in any browser. Messages will appear in real time in the tab.

## Using the CLI

### Build

```bash
git clone <this-repo>
cd nfty-test
go build -o nfty-test .
```

### Basic usage

The tool reads from stdin and sends each line as a notification:

```bash
echo "deploy complete" | ./nfty-test -topic myteam-alerts-2026
```

### Send with a title and priority

```bash
echo "prod is down" | ./nfty-test -topic myteam-alerts-2026 \
  -title "ALERT" \
  -priority max \
  -tags rotating_light
```

Priority levels: `min`, `low`, `default`, `high`, `max`

### Pipe git commits

Each commit is sent as a separate notification. Tapping the notification opens the commit on GitHub or Gitea (auto-detected from the git remote):

```bash
git log --oneline -5 | ./nfty-test -topic myteam-alerts-2026 -repo /path/to/repo
```

### Batch multiple lines into one notification

```bash
git log --oneline -5 | ./nfty-test -topic myteam-alerts-2026 \
  -batch \
  -repo /path/to/repo \
  -title "Recent Commits"
```

### Clickable notifications

Manually set a URL that opens when the notification is tapped:

```bash
echo "PR ready for review" | ./nfty-test -topic myteam-alerts-2026 \
  -click https://github.com/org/repo/pull/42
```

When piping `git log --oneline`, the click URL is set automatically — no `-click` flag needed. The tool reads the commit hash from each line and the remote URL from the repo to build the correct link.

### Action buttons

Add buttons to the notification:

```bash
echo "build failed" | ./nfty-test -topic myteam-alerts-2026 \
  -actions "view, Open CI, https://ci.example.com/build/123; view, View Logs, https://ci.example.com/logs/123"
```

### Chain with other tools

The tool echoes messages to stdout, so it works in pipelines:

```bash
git log --oneline -3 | ./nfty-test -topic myteam-alerts-2026 -repo . | tee notification-log.txt
```

### All flags

| Flag | Description | Default |
|------|-------------|---------|
| `-topic` | Topic name to publish to (required) | |
| `-server` | ntfy server URL | `https://ntfy.sh` |
| `-title` | Notification title | |
| `-priority` | `min`, `low`, `default`, `high`, `max` | |
| `-tags` | Comma-separated emoji tags | |
| `-click` | URL to open on tap | auto-detected from git |
| `-actions` | Action buttons | |
| `-batch` | Combine all stdin into one notification | `false` |
| `-repo` | Path to git repo for commit URL resolution | `.` |

## Self-hosting

If you need more than 250 messages/day or want private topics, ntfy is open source and can be self-hosted. Point the `-server` flag at your instance:

```bash
echo "hello" | ./nfty-test -topic alerts -server https://ntfy.internal.example.com
```
