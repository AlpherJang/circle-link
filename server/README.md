# Server Module

The `server` module is responsible for:

- authentication and session issuance
- device registration
- public key bundle storage
- online presence
- WebSocket-based ciphertext relay
- encrypted offline mailbox delivery

The server must not persist plaintext or permanent chat message history. To support offline delivery, it may temporarily store encrypted undelivered message envelopes with TTL-based cleanup.

Recommended internal layout:

```text
server/
|-- cmd/
|   |-- api/
|   `-- relay/
|-- configs/
|-- internal/
|   |-- auth/
|   |-- device/
|   |-- keys/
|   |-- presence/
|   |-- relay/
|   `-- store/
|-- migrations/
`-- pkg/
```
