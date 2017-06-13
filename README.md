Mastodon slack integration

```yml
serverConfs:
  - serverURL: <mastodon rails server url>
    streamingServerURL: <mastodon streaming(websocket) server url>
    clientID:  <oauth client id>
    clientSecret: <oauth client secret>
    account: <account>
    password: <password>
slackWebHookURL: <slack webhook url>
```

# Preparation for using Mastodon API

Requests the mastdon server to issue a client ID / secret.
Call curl as follows.
Please modify the contents as appropriate.

```bash
curl -X POST -sS https://xxxxxxxxxxxxxx/api/v1/apps \
   -F "client_name=xxxxxxxxxx" \
   -F "redirect_uris=urn:ietf:wg:oauth:2.0:oob" \
   -F "scopes=read write follow"
```

