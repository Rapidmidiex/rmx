import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

// https://vitejs.dev/config/
export default defineConfig({
    build: {
        lib: {
            entry: ["./src/components/PianoNote.svelte"],
            fileName: 'bundle',
            formats: ['es' /* ,'umd' */]
        }
    },
    plugins: [svelte({
        compilerOptions: {
            customElement: true
        }
    })],
});
