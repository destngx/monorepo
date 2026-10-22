<script lang="ts">
  import ChatPanel from '$lib/chat/ChatPanel.svelte';
  import { Card } from '@libs/ui-kit';
  import PageHeader from '$lib/components/PageHeader.svelte';
</script>

<!-- WM-AI-11 -->
<PageHeader title="Assistant" subtitle="Answers use your live sheet data through read-only tools" />

<Card class="flex h-[calc(100dvh-19rem)] min-h-96 flex-col lg:h-[calc(100vh-14rem)]">
  <ChatPanel page={{ path: '/chat', title: 'Assistant' }} class="h-full" />
</Card>
