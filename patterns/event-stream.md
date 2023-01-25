# Event Stream


Event Sourcing is a pattern for storing data as events in an append-only log. In constrast with state-oriented databases that only keeps the latest version of the entity state, you can store each state change as a separate event [^1].


Most of the time, we don't need a full-fledged solution for storing events, and any database could be used to persist those events.

## Streams

Events are logically grouped into streams. They are the representation of the entities.

1. For the tables that requires persisting the events, we can suffix those tables with `_streams` (not `_events`).
2. The events can be persisted atomically in a single transaction, storing both the final state of the entity as well as the events leading to that state.
3. Implementing optimistic concurrency can be tricky.

See the other repo for the actual implementation.


[^1]: https://developers.eventstore.com/server/v5/streams.html#metadata-and-reserved-names:~:text=In%20contrast%20with%20state%2Doriented%20databases%20that%20only%20keeps%20the%20latest%20version%20of%20the%20entity%20state%2C%20you%20can%20store%20each%20state%20change%20as%20a%20separate%20event.
[^2]: https://developers.eventstore.com/server/v5/streams.html#metadata-and-reserved-names:~:text=Events%20are%20logically%20grouped%20into%20streams.%20They%20are%20the%20representation%20of%20the%20entities.
