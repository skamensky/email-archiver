const startWSConnection = () => {

    // const webServer = `ws://${window.location.hostname}:${window.location.port}/ws`;
    const webServer = `ws://localhost:8080/ws`;
    var ws = new WebSocket(webServer);
    ws.onopen = function () {
        ws.send("Hello, world");
    };
    ws.onmessage = function (event) {
        var msg = event.data;
        try {
            msg = JSON.parse(msg);
        } catch (e) {
        }
        console.log(msg);
    };
    ws.onclose = function () {
        console.log("Connection is closed...");
    }
    return ws;
}
export {
    startWSConnection
}