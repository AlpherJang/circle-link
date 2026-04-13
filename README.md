# circle-link

`circle-link` is a private chat system built around two core modules:

1. `server`: authentication, device registration, encrypted session bootstrap, real-time relay, presence, and offline encrypted mailbox delivery. The server does not persist plaintext or permanent chat history.
2. `client`: a shared cross-platform application that targets iOS, macOS, and Android from one main codebase.

The initial repository layout is:

```text
.
|-- client/
|-- docs/
|-- protocol/
`-- server/
```

The primary technical design lives in [docs/technical-design.md](docs/technical-design.md).
The implementation-facing model and interface design lives in [docs/model-and-api-design.md](docs/model-and-api-design.md).

## Local Debug Messaging

The current server bootstrap includes a local debug page at `/debug` for smoke-testing:

- sign up
- verify email with the dev token returned by signup
- log in
- register a device
- send and receive messages between two browser tabs
- receive inbox updates live through a debug SSE stream at `/v1/messages/stream`
- open a websocket session with `session.bind`
- emit recipient delivery acknowledgements with `message.ack`
- send message payloads using envelope-like fields such as `messageId`, `conversationId`, `contentType`, and `clientMessageSeq`

This path is intended only for local development. It does not represent the final end-to-end encrypted client flow.
