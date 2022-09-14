export function websocketUrl(path) {
    var url = new URL(path, window.location.href);

    url.protocol = url.protocol.replace('http', 'ws');

    return url.href; // => ws://www.example.com:9999/path/to/websocket
};