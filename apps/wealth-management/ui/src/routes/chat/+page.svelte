<script lang="ts">
  import ChatPanel from '$lib/chat/ChatPanel.svelte';
  import { Card } from '@libs/ui-kit';
  import PageHeader from '$lib/components/PageHeader.svelte';
</script>

<!-- WM-AI-11 -->
<PageHeader title="Assistant" subtitle="Answers use your live sheet data through read-only tools" />

<!-- On phones the card ends above the area bar: the screen less the header (under the notch), the page title
     and the bar. -->
<Card
  class="flex h-[calc(100dvh-var(--spacing-tabbar)-max(0.75rem,env(safe-area-inset-top))-11.5rem)] min-h-96 flex-col lg:h-[calc(100vh-14rem)]"
>
  <ChatPanel page={{ path: '/chat', title: 'Assistant' }} class="h-full" />
</Card>
