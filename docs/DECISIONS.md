# Decisions LOG

Given the time constraints. I'll keep it as simple as possible.

## Assumptions

Since I assume that it is more interesting for this test the out-of-order events handling and the quality of code I
decided to use the `oapi-codegen` library to generate API code and focus on the business logic.

## Infrastructure

I'll use mongodb as a single dependency.

## Design

The initial idea is to have a messages log. After a message is received the app
will persist the message to the log always, after it will check if all the messages for that channel are in sequence and
upsert the rocket with the latest one, if so, it will apply all the messages for that channel from the log,
a resequencer approach. I was thinking for some time and decided to go with this approach for simplicity. The other
option was to have a buffer just with the messages out of order and apply them over snapshots, it would be more
efficient but a little more complex, so i decided to start with something.

## Trade-offs

Testing: I decided to write a single end-to-end test to check the happy path, and later make it pass.

Code structure: Normally I use a more explicit hexagonal architecture approach but for the sake of simplicity I decided
simplify module structure and for example keep the infra code in the same file as the interfaces.

Magic strings, numbers and hardcoded values: in a production ready app some stuff should be extracted to config files.

Poor error modeling: More errors can be modeled

Input validation: for a production ready app I would add input validation like negative speed...


## Future improvements
Input validation.

Moving to buffer approach to optimize memory and performance. Use redis as a buffer.

Make the processing async to remove latency from the request.
