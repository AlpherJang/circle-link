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
