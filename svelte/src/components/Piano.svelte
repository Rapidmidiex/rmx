<script lang="ts">
    import Key from './Key.svelte';
    import { Note, action } from '../lib/piano';

    // bindings go here
    const map = new Map<string, { bind: string; note: Note }>([
        // don't be scared use enum
        ['s', { bind: 's', note: Note.C }],
        ['d', { bind: 'd', note: Note.D }],
        ['f', { bind: 'f', note: Note.E }],
        ['h', { bind: 'h', note: Note.F }],
        ['j', { bind: 'j', note: Note.G }],
        ['k', { bind: 'k', note: Note.A }],
        ['l', { bind: 'l', note: Note.B }],
        ['e', { bind: 'e', note: Note.Df }],
        ['r', { bind: 'r', note: Note.Ef }],
        ['u', { bind: 'u', note: Note.Gf }],
        ['i', { bind: 'i', note: Note.Af }],
        ['o', { bind: 'o', note: Note.Bf }],
    ]);
</script>

<svelte:window use:action={{ map }} />

{#each map as [_, props]}
    <Key {...props} />
{/each}
