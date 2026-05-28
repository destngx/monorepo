# VOZ Crawl Strategy

This document captures the current site-specific crawl strategy for `voz.vn`.

## Purpose

- Crawl VOZ thread pages through the browser service.
- Extract only the post content needed for downstream use.
- Persist checkpointed progress so long threads can be resumed.
- Keep VOZ-specific rules isolated from shared crawler plumbing.

## Scope

- Applies only to the VOZ site module at `src/modules/sites/voz/`.
- Does not define the generic browser probe flow.
- Does not cover other sites.

## Crawl Model

- Input is a VOZ thread URL.
- The crawler resolves the VOZ thread ID from the URL.
- The crawler determines the last page from the rendered XenForo HTML.
- Pages are fetched sequentially from page 1 through the last page or an optional `max_pages` cap.
- Each page is parsed into replies and appended to the thread output.

## Output Shape

- Results are stored under `data/crawls/<domain>/<thread_id>/`.
- The main output file is `<thread_id>.json`.
- The checkpoint file is `<thread_id>.checkpoint.json`.
- The crawl output is nested by thread, page, and reply:
  - thread
  - pages
  - replies
- Reply records only keep:
  - `author`
  - `created_at`
  - `text`

## VOZ-Specific Behavior

- Page URLs use VOZ pagination rules.
- Post extraction is based on XenForo thread markup.
- Page completion is detected from actual posts in the HTML, not only from HTTP status.
- `403 blocked` is treated as a recoverable session/profile failure.
- On first `403 blocked`, the crawler writes a checkpoint and retries once with a fresh browser profile.
- If the retry fails, the crawl stops and preserves the checkpoint.
- Jittered pacing is used between page fetches instead of a fixed interval.
- `resume_from_page` resumes the crawl from the requested page or the checkpointed page.

## Shared Dependencies

- Browser acquisition and profile management come from shared browser services.
- HTML challenge classification comes from shared detector helpers.
- Checkpoint/output path resolution comes from shared crawl storage helpers.
- The checkpoint payload is a shared typed model.

## Current Limits

- The strategy is optimized for VOZ thread pages only.
- It assumes the thread layout stays consistent with the current XenForo HTML structure.
- It does not yet generalize thread parsing rules to other forum engines or content types.
