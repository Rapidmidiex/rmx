import { websocketUrl, sessionId } from "./utils.js";

const app = document.getElementById("app");

app.innerHTML = `
    <h1>Welcome to the JAM Session</h1>
    <button disabled>Say Hello</button>
    
    <ul aria-label="users"></ul>
`;

const ws = new WebSocket(websocketUrl(`/jam/${sessionId()}`));

ws.addEventListener("open", e => {
    document.querySelector("button").disabled = false;

    alert("web socket has opened");
});

ws.addEventListener("message", async e => {
    const { type, id } = JSON.parse(e.data);

    switch (type.toLowerCase()) {
        case "join":
            newUserJoined();
            break;
        case "leave":
            document.getElementById(id).remove();
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

async function newUserJoined() {
    const r = await fetch(`/api/jam/${sessionId()}`);
    const { users } = await r.json();

    // console.log(users);

    // ^proxy Array that updates the list in the DOM
    const items = [];
    for (const id of users) {
        const li = document.createElement("li");
        li.textContent = `${id} has joined`;
        li.id = id;
        items.push(li);
    }

    document
        .querySelector(`[aria-label="users"]`)
        .replaceChildren(...items);
}