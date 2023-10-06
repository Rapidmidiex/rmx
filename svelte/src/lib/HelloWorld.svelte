<svelte:options customElement="hello-world" />

<!-- in Svelte 3, do this instead:
<svelte:options tag="my-element" />
-->

<script>
	export let name = 'world';
</script>

<h1>Hello {name}!</h1>
<slot />