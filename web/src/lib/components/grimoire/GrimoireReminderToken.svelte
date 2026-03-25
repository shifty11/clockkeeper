<script lang="ts">
  import { Team } from "~/lib/gen/clockkeeper/v1/clockkeeper_pb";
  import {
    teamCardColors,
    goodColors,
    evilColors,
    teamDataAttr,
  } from "~/lib/team-styles";
  import type { GrimoireReminder } from "./types";

  let {
    reminder,
    zoom,
    onmove,
    ondragmove,
  }: {
    reminder: GrimoireReminder;
    zoom: number;
    onmove?: (x: number, y: number) => void;
    ondragmove?: (x: number, y: number) => void;
  } = $props();

  // Effective alignment for icon/color (alignment override takes priority)
  const effectiveAlignment = $derived<"good" | "evil" | undefined>(
    reminder.alignment ??
      (reminder.team === Team.TOWNSFOLK || reminder.team === Team.OUTSIDER
        ? "good"
        : reminder.team === Team.MINION || reminder.team === Team.DEMON
          ? "evil"
          : undefined),
  );

  const iconSuffix = $derived(
    effectiveAlignment === "good"
      ? "_g"
      : effectiveAlignment === "evil"
        ? "_e"
        : "",
  );
  const iconUrl = $derived(
    `/characters/${reminder.edition}/${reminder.characterId}${iconSuffix}.webp`,
  );

  const colorClass = $derived(
    reminder.alignment
      ? reminder.alignment === "good"
        ? goodColors
        : evilColors
      : (teamCardColors[reminder.team] ?? "border-border bg-surface-alt"),
  );

  const isAttached = $derived(!!reminder.attachedTo);

  let dragging = $state(false);
  let dragStartX = $state(0);
  let dragStartY = $state(0);
  let offsetX = $state(0);
  let offsetY = $state(0);
  let imgError = $state(false);

  function onPointerDown(e: PointerEvent) {
    e.stopPropagation();
    dragging = true;
    dragStartX = e.clientX;
    dragStartY = e.clientY;
    offsetX = 0;
    offsetY = 0;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  }

  function onPointerMove(e: PointerEvent) {
    if (!dragging) return;
    offsetX = (e.clientX - dragStartX) / zoom;
    offsetY = (e.clientY - dragStartY) / zoom;
    ondragmove?.(reminder.x + offsetX, reminder.y + offsetY);
  }

  function onPointerUp() {
    if (!dragging) return;
    dragging = false;
    if (offsetX !== 0 || offsetY !== 0) {
      onmove?.(reminder.x + offsetX, reminder.y + offsetY);
    }
    offsetX = 0;
    offsetY = 0;
  }
</script>

<div
  class="absolute touch-none select-none"
  style="left: {reminder.x + offsetX}px; top: {reminder.y +
    offsetY}px; transform: translate(-50%, -50%); z-index: {dragging ? 50 : 0};"
  onpointerdown={onPointerDown}
  onpointermove={onPointerMove}
  onpointerup={onPointerUp}
  onpointercancel={onPointerUp}
  role="button"
  tabindex="0"
>
  <div
    class="card-slate token-bezel-sm flex h-16 w-16 flex-col items-center justify-center rounded-full p-0.5 {colorClass} {isAttached && !dragging ? 'ring-2 ring-primary/20' : ''}"
    data-team={teamDataAttr[reminder.team] ?? ""}
  >
    {#if !imgError && reminder.edition}
      <img
        src={iconUrl}
        alt={reminder.characterName}
        class="h-10 w-10 shrink-0 drop-shadow-sm"
        onerror={() => (imgError = true)}
        draggable="false"
      />
    {:else}
      <div
        class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-element text-sm text-secondary"
      >
        {reminder.characterName.charAt(0)}
      </div>
    {/if}
  </div>
  <div
    class="mt-0.5 max-w-16 text-center text-[10px] leading-tight text-secondary"
  >
    {reminder.text}
  </div>
</div>
