const app = document.getElementById("app");

app.innerHTML = `
    <h1>This is a dynamic page</h1>
    <button>Click Me!</button>
`;

const ws = new WebSocket(`ws://localhost:8888/jam`);

ws.addEventListener("open", e => {
    alert("web socket has opened");
});

ws.addEventListener("message", e => {
    alert(e.data);
});

ws.addEventListener("error", e => {
    console.error(e);

    document.querySelector("button").disabled = true;
});

document.querySelector("button").addEventListener("click", e => {
    ws.send(1);
});