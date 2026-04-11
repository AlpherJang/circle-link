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

Suggested layout:

```text
client/
|-- apps/
|-- lib/
|-- native/
|-- rust/
`-- test/
```
