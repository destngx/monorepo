<script lang="ts">
  // How to photograph an imaging film (HW-IMP-7). Film photos are kept for
  // reference and never read with AI; the written report is the part to file.
  let { open = false }: { open?: boolean } = $props();
</script>

<details class="rounded-xl bg-slate-900/5 px-3 py-2 text-left text-sm dark:bg-white/5" {open}>
  <summary class="cursor-pointer font-medium">How to photograph an X-ray or CT film</summary>
  <ol class="muted mt-2 list-decimal space-y-1 pl-5">
    <li>Hold the film against a bright window, or a laptop or TV screen showing a white page at full brightness.</li>
    <li>Turn the flash off and the room lights down, so nothing reflects on the film.</li>
    <li>Hold the phone straight above the film, with the whole film in the frame and its edges straight.</li>
    <li>Tap the film on the screen to focus, wait until it is sharp, then take the photo.</li>
    <li>
      Upload the written radiology report too. It is the part that is read and searched; the film photo is kept to look
      at.
    </li>
  </ol>
</details>
