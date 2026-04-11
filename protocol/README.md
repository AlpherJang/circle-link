# Protocol Module

The `protocol` module owns shared contracts used by both server and client:

- protobuf message schemas
- relay event envelopes
- mailbox delivery envelopes
- API DTO versioning rules
- error code conventions

Suggested layout:

```text
protocol/
`-- proto/
```

All message and event contracts should be defined here first and then generated into server and client language bindings.
