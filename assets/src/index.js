import { websocketUrl } from "./utils.js";

const app = document.getElementById("app");

app.innerHTML = `
    <h1>Home</h1>
    <p>To create a new session tap the button below</p>
    <button>Click Me!</button>
`;

document.querySelector("button").addEventListener("click", async e => {
    try {
        const r = await fetch("/api/jam/create");
        const { sessionId } = await r.json();
        console.log(sessionId);
    } catch (e) {
        console.error(e.message);
    }
});
