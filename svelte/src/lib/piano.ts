import { writable, type Updater } from 'svelte/store';

// is a map
export const counter  =writable(0);

export const keyMap = writable<Map<string, {
    bind: string, note: Note
}>>()

export const key = Symbol();

const frequencyFromNote = (n: number, base = 440) => {
    // using A as the base
    return Math.pow(2, n / 12) * base;
};

export enum Note {
    A,
    Bf,
    B,
    C,
    Df,
    D,
    Ef,
    E,
    F,
    Gf,
    G,
    Af,
}


export function play(note: number) {
    const audioContext = new AudioContext();

    // custom is allowed
    const oscillator = new OscillatorNode(audioContext, {
        frequency: frequencyFromNote(note),
        type: 'sine',
        // detune:
    });

    oscillator.connect(audioContext.destination);
    oscillator.start();
    oscillator.stop(audioContext.currentTime + 1);
}

export function action(
    el: Window,
    {map}: { map: Map<string, { bind: string; note: Note }> }
) {
    function onKeyDown(e: KeyboardEvent) {
        if (
            map.has(e.key) &&
            // this is just fancy, don't copy
            !e.repeat &&
            !void e.preventDefault()
        ) {
            // NOTE -- TS cannot infer this yet
            const { note } = map.get(e.key)!;

            play(note);
        }
    }

    window.addEventListener('keydown', onKeyDown);
    return {
        destroy() {
            window.removeEventListener('keydown', onKeyDown);
        },
        // update(opts)
    };
}