var bw2lib = (function () {
    var Client = function(key) {
        this.key = key;
        this._callbacks = {};
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

    return {
        Client: Client
    };
    
}());
