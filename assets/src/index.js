import { websocketUrl } from "./utils.js";

const app = document.getElementById("app");

app.innerHTML = `
    <h1>This is a dynamic page</h1>
    <button disabled>Click Me!</button>

    <ul aria-label="users"></ul>
`;

const ws = new WebSocket(websocketUrl("/jam"));

ws.addEventListener("open", e => {
    document.querySelector("button").disabled = false;

    alert("web socket has opened");
});

ws.addEventListener("message", async e => {
    const { type, id } = JSON.parse(e.data);

    switch (type.toLowerCase()) {
        case "join":
            userJoinedSession({ id });
            break;
        case "leave":
            userLeftSession({ id });
            break;
    }
});

ws.addEventListener("error", e => {
    console.error(e);

    document.querySelector("button").disabled = true;
});

document.querySelector("button").addEventListener("click", e => {
    ws.send(1);
});

function userJoinedSession({ id }) {
    const li = document.createElement("li");
    li.textContent = `${id} has joined`;
    li.id = id;

    document.querySelector(`[aria-label="users"]`).appendChild(li);
}

function userLeftSession({ id }) {
    const li = document.getElementById(id);
    li.remove();
}