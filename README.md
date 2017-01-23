# BOSSWAVE Proxy

For local "offline" web apps. Idea is to allow BOSSWAVE application/UI development using HTML/CSS/JS, talking to a local web proxy to communicate over BOSSWAVE

## Design Discussion

We may actually want to do a set of more involved actions, including:
- subscribing:
    - long running connection. Probably web-sockets?
- long-running async operations:
    - e.g. wait for a db query to complete
- regular request-response:

Will want a client library that is loaded from the proxy server that gives you all the function/library calls you need.
Then we can change the implementation behind it whenever we want.

Implement library in JS; takes care of the "RPC" aspect of it. Don't really want to use JSON-RPC

What's the implementation?
- how to "call" functions that are bosswave api calls
- how to limit what an app can do
- localstorage api?
- addon libraries for remote services e.g. archiver

## Implementation

### Endpoints:
- have one endpoint for each 'type' of call:
    - request/response
    - streaming (websocket)
    - others??
- receive json object with function call + params:
    - deserialize to actual Params struct (e.g. bw2bind subscribe params)
    - do the call
    - take result, deserialize to interface{}, then serialize to json
    - send result back

### Permissions:
- you will register your entity w/ the trusted bwproxy; you will get back an API key with a set of permissions.
- give this API key to the application (used in the JS library)



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
