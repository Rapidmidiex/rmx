<svelte:options customElement="piano-note" />

<script>
    const audioContext = new AudioContext();

    function playNote() {
        const oscillator = audioContext.createOscillator();

        oscillator.type = 'square';
        oscillator.frequency.setValueAtTime(440, audioContext.currentTime); // value in hertz
        oscillator.connect(audioContext.destination);
        oscillator.start();
        // 2s
        oscillator.stop(audioContext.currentTime+2)
    }
</script>

<button on:click={playNote}>play</button>
