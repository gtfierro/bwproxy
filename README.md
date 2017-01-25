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

What's the implementation?
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

### Registering/Onboarding

Right now its a command-line tool, but that's not as nice to use

- admin dashboard:
    - use an internal API on a DIFFERENT server:
        - avoids untrusted apps accessing admin privileges by using same-origin policy
    - browse/install/uninstall apps
    - other configuration
- local DNS record:
    - append to /etc/hosts, probably
    - can we redirect the port? Or just get it to listen on port 80?

### Installing Apps

- apps are JUST a bundle of html/js/css pages, plus a config file
- config file is just JSON that the app loads:
    - app loads the config by requesting it from the bw2proxy server
- config contents:
    - key to use

### Application Structure

- index.html file
- static files
- manifest.json:
    - descriptions about the application: name + description
    - desired domain name? Need some local DNS for this
