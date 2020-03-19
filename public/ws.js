// WebSocket objects - created when window is loaded.
var sockEcho = null;
var sockTime = null;

// Websocket server address.
var wsServerAddress = "ws://127.0.0.1:4050";

window.onload = function() {
    // Connect the WebSocket to the server and register callbacks on it.
    sockEcho = new WebSocket(wsServerAddress + "/wsecho");

    sockEcho.onopen = function() {
        console.log("connected");
    }

    sockEcho.onclose = function(e) {
        console.log("connection closed (" + e.code + ")");
    }

    sockEcho.onmessage = function(e) {
        var msg = JSON.parse(e.data);
        var coordMsg = "Coordinates: (" + msg.x + "," + msg.y + ")";
        document.getElementById("output").innerHTML = coordMsg;
    }

    // This is a pure push notification from the server: register onmessage
    // to update the time when the server sends new timestamps.
    sockTime = new WebSocket(wsServerAddress + "/wstime");
    sockTime.onmessage = function(e) {
        document.getElementById("timeticker").innerHTML = e.data;
    }
};

// Send the msg object, encoded with JSON, on the websocket if it's open.
function socketSend(msg) {
    if (sockEcho != null && sockEcho.readyState == WebSocket.OPEN) {
        sockEcho.send(JSON.stringify(msg));
    } else {
        console.log("Socket isn't OPEN");
    }
}

function onMouseMoved(e) {
    // When a "mouse moved" event is invoked, send it on the socket.
    socketSend({x: e.clientX, y: e.clientY});
}

function onMouseOut() {
    document.getElementById("output").innerHTML = "";
}
