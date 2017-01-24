var bw2lib = (function () {
    var Client = function(key) {
        this.key = key;
        this._subscriptions = {};
    };

    Client.prototype.query = function(params, success, failure) {
        var params = {
            key: this.key,
            proc: "query",
            params: params
        };
        console.log("QUERY",params);
        $.post("/call", JSON.stringify(params))
            .done(function(data) {
                success(data);
            })
            .fail(function(err) {
                failure(err);
            });
    };

    Client.prototype.publish = function(params, success, failure) {
        var params = {
            key: this.key,
            proc: "publish",
            params: params
        };
        $.post("/call", JSON.stringify(params))
            .done(function(data) {
                success(data);
            })
            .fail(function(err) {
                failure(err);
            });
    };

    Client.prototype.subscribe = function(params, success, failure) {
        var ws = new WebSocket("ws://"+window.location.host+"/streaming");
        var params = {
            key: this.key,
            proc: "subscribe",
            params: params
        };
        ws.onmessage = function(e) {
            success(JSON.parse(e.data));
        }
        ws.onerror = function(e) {
            failure(e.data)
        }
        ws.onopen = function(e) {
            console.log("OPEN")
        ws.send(JSON.stringify(params));
        }
    };

    return {
        Client: Client
    };
    
}());
