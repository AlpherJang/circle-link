# Client Module

The `client` module is the shared application stack for:

- iOS
- macOS
- Android

Recommended stack:

- Flutter for UI and application logic
- Rust for shared crypto
- secure OS storage for keys and tokens
- encrypted local database for on-device chat history
- default persistent local history plus optional disappearing messages

Current local app status:

- sign up
- email verification with the dev token returned by the server
- login and logout
- device registration, selection, and revocation
- websocket session binding for the active device
- loading and inviting contacts through the server contact API
- accepting or declining pending contact invites
- loading conversation summaries through the server conversation API
- loading device-filtered inbox messages
- sending debug chat messages through websocket with HTTP fallback
- merging server conversation summaries with local live message state
- auto-emitting `delivered` receipts for live messages
- tapping inbox messages to emit `read` receipts
- sender-side status updates for `accepted`, `delivered`, and `read`

Current limitation:

- this is still a debug client flow wired to the local relay API and not the final Rust-backed E2EE client
- conversation summaries are still message-derived on the server, not yet a dedicated persisted conversation model
- tokens are stored in memory only for now
- there is no persistent local database yet

Suggested layout:

```text
client/
|-- apps/
|-- lib/
|-- native/
|-- rust/
`-- test/
```
