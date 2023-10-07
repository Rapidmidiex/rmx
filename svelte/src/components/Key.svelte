<script lang="ts">
    import { onMount } from 'svelte';
    import { Note, counter, keyMap, play } from '../lib/piano';

    export let note: Note;
    export let bind: string;

    // onMount(() => {
    //     keyMap.update(
    //         ($keyMap) => new Map([...($keyMap??new Map()), [bind, { bind, note }]])
    //     );
    // });

    // add to store from context?
    // key->element
    // w/element->key
</script>

<button on:click={() => play(note)}> {bind}/{'todo'} </button>
