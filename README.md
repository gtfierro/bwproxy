# BOSSWAVE Proxy

For local "offline" web apps. Idea is to allow BOSSWAVE application/UI development using HTML/CSS/JS, talking to a local web proxy to communicate over BOSSWAVE

## URI Endpoints

GET `/sub/<uri>`, `/subscribe/<uri>`:
- delivers JSON messages of what's published on BOSSWAVE (websockets)

GET `/query/<uri>`:
- returns JSON doc of persisted message on the URI

GET `/latest/<uri>`:
- latest message published on this URI
- application can 'request' a set of URIs to be subscribed to on the backend

POST `/register`:
- request backend subscriptions on the URI (list of URIs)
- specify the entity, other params for operation
- get back an API key?

POST `/heartbeat/<api key>`:
- keep backend subscriptions alive?

POST `/pub/<uri>`, `/publish/<uri>`:
- publish message on URI. Takes JSON, converts to msgpack or whatever

## Implementation

Go server to take advantage of bw2bind

Need local database to store messages? We can probably just query the bw2 agent UNLESS it crashes/goes offline? Do we want to be independent of that?
