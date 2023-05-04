# RedactedHook

RedactedHook is a webhook companion service for [autobrr](https://github.com/autobrr/autobrr) designed to check name of uploaders and your ratio on Redacted. It provides a simple and efficient way to validate if an uploader is blacklisted or if you want to stop racing in case your ratio falls below a certain point.

## Features

- Verify if an uploader's name is on a provided blacklist
- Check if a user's ratio meets a specified minimum value
- Easy to integrate with other applications via webhook

It was made with [autobrr](https://github.com/autobrr/autobrr) in mind.

## Getting Started

### Prerequisites

To run RedactedHook, you'll need:

1. Go 1.20 or later installed (only if building from source)
2. Access to Redacted

### Installation

#### Docker

```bash
docker pull ghcr.io/s0up4200/redactedhook:latest
```

**docker compose**

```docker
version: "3.7"
services:
  redactedhook:
    container_name: redactedhook
    image: ghcr.io/s0up4200/redactedhook:latest
    user: 1000:1000
    #environment:
    #  - SERVER_ADDRESS=0.0.0.0 # binds to 127.0.0.1 by default
    #  - SERVER_PORT=42135 # defaults to 42135
    ports:
      - "42135:42135"
```

#### Using precompiled binaries

Download the appropriate binary for your platform from the [releases](https://github.com/s0up4200/RedactedHook/releases/latest) page.

#### Building from source

1. Clone the repository:

```bash
git clone https://github.com/s0up4200/RedactedHook.git
```

2. Navigate to the project directory:

```bash
cd RedactedHook
```
3. Build the project:

```go
go build
```
or
```shell
make build
```

4. Run the compiled binary:

```bash
./bin/RedactedHook
```

### Usage

To use RedactedHook, send POST requests to the following endpoints:

#### Check Ratio

- Endpoint: `http://127.0.0.1:42135/redacted/ratio`
- Method: POST
- Expected HTTP Status: 200

**JSON Payload:**

```json
{
  "user_id": USER_ID,
  "apikey": "API_KEY",
  "minratio": MINIMUM_RATIO
}
```

`user_id` is the number in the URL when you visit your profile.

`api_key` your Redacted API key. Needs user and torrents privileges.

#### Check Uploader

- Endpoint: `http://127.0.0.1:42135/redacted/uploader`
- Method: POST
- Expected HTTP Status: 200

**JSON Payload:**

```json

{
  "torrent_id": {{.TorrentID}},
  "apikey": "API_KEY",
  "uploaders": "BLACKLISTED_USER1,BLACKLISTED_USER2,BLACKLISTED_USER3"
}
```

`torrent_id` will automatically be filled when you use `{{.TorrentID}}` - a macro supported by autobrr.

`api_key` your Redacted API key. Needs user and torrents privileges.

#### curl commands for easy testing

```bash
curl -X POST -H "Content-Type: application/json" -d '{"user_id": 3855, "apikey": "e1be0c8f.6a1d6f89de6e9f6a61e6edcbb6a3a32d", "minratio": 1.0}' http://127.0.0.1:42135/redacted/ratio
```
```bash
curl -X POST -H "Content-Type: application/json" -d '{"torrent_id": 3931392, "apikey": "e1be0c8f.6a1d6f89de6e9f6a61e6edcbb6a3a32d", "uploaders": "blacklisted_user1,blacklisted_user2,blacklisted_user3"}' http://127.0.0.1:42135/redacted/uploader
