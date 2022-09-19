const app = document.getElementById('app');

app.innerHTML = `
    <h1>Welcome, Home</h1>
    <p>To create a new session tap the button below</p>
    <button>Click Me!</button>

    <div id="session"></div>
`;

document.querySelector('button').addEventListener('click', async (e) => {
    try {
        const r = await fetch('/api/v1/jam', {
            method: 'POST',
            body: {
                // TODO: Add any info from UI
            },
        });
        const { sessionId } = await r.json();

        session.id = sessionId;
    } catch (e) {
        console.error(e.message);
    }
});

const session = new Proxy(
    { id: '' },
    {
        set(obj, prop, value) {
            let v = 0;
            switch (prop) {
                case 'id':
                    obj[prop] = value;
                    const url = `${window.location.href}play/${value}`;
                    const anchor = document.createElement('a');
                    anchor.href = url;
                    anchor.innerText = url;
                    anchor.target = '_blank';
                    document.getElementById('session').appendChild(anchor);

                    return true;

                default:
                    return false;
            }
        },
    }
);
