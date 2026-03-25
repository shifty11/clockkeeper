<script lang="ts">
  import { page } from "$app/state";
  import type { Character } from "~/lib/gen/clockkeeper/v1/clockkeeper_pb";
  import { formatReminder } from "~/lib/format";
  import { teamLabels, teamBadgeColors, iconSuffix } from "~/lib/team-styles";
  import ReminderToken from "./ReminderToken.svelte";

  let {
    character,
    onclose,
    onstartgame,
    canStartGame = false,
    scriptId,
  }: {
    character: Character;
    onclose: () => void;
    onstartgame?: () => void;
    canStartGame?: boolean;
    scriptId?: bigint;
  } = $props();

  const suffix = $derived(iconSuffix(character.team));
  const iconUrl = $derived(
    `/characters/${character.edition}/${character.id}${suffix}.webp`,
  );

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") onclose();
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
  onclick={onclose}
  onkeydown={handleKeydown}
>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="mx-4 w-full max-w-lg rounded-xl border border-border bg-surface p-6 shadow-2xl"
    onclick={(e) => e.stopPropagation()}
  >
    <!-- Header with icon and name -->
    <div class="flex items-center gap-4">
      <img
        src={iconUrl}
        alt={character.name}
        class="h-20 w-20 shrink-0 rounded-full"
        onerror={(e: Event) =>
          ((e.target as HTMLImageElement).style.display = "none")}
      />
      <div class="min-w-0 flex-1">
        <h3 class="text-xl font-bold text-primary">{character.name}</h3>
        <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
          <span
            class="rounded-full px-2.5 py-0.5 text-xs font-medium {teamBadgeColors[
              character.team
            ] ?? 'bg-element text-muted'}"
          >
            {teamLabels[character.team] ?? "Unknown"}
          </span>
          {#if character.setup}
            <span
              class="rounded-full bg-yellow-100 px-2.5 py-0.5 text-xs font-medium text-yellow-700 dark:bg-yellow-500/20 dark:text-yellow-300"
              >setup</span
            >
          {/if}
        </div>
      </div>
      <button
        onclick={onclose}
        class="shrink-0 rounded-lg p-1.5 text-muted transition-colors hover:bg-hover hover:text-primary"
        aria-label="Close"
      >
        <svg
          class="h-5 w-5"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>

    <!-- Ability -->
    <p class="mt-5 text-sm leading-relaxed text-secondary">
      {character.ability}
    </p>

    <!-- Night actions -->
    {#if character.firstNight > 0 || character.otherNight > 0}
      <div class="mt-4 space-y-2">
        {#if character.firstNight > 0 && character.firstNightReminder}
          <div class="flex items-start gap-2 text-xs">
            <span
              class="shrink-0 rounded bg-amber-100 px-1.5 py-0.5 font-medium text-amber-700 dark:bg-amber-500/20 dark:text-amber-300"
              >1st Night</span
            >
            <span class="text-secondary"
              >{@html formatReminder(character.firstNightReminder)}</span
            >
          </div>
        {/if}
        {#if character.otherNight > 0 && character.otherNightReminder}
          <div class="flex items-start gap-2 text-xs">
            <span
              class="shrink-0 rounded bg-indigo-100 px-1.5 py-0.5 font-medium text-indigo-700 dark:bg-indigo-500/20 dark:text-indigo-300"
              >Other</span
            >
            <span class="text-secondary"
              >{@html formatReminder(character.otherNightReminder)}</span
            >
          </div>
        {/if}
      </div>
    {/if}

    <!-- Reminder tokens with images -->
    {#if character.reminders.length > 0 || character.remindersGlobal.length > 0}
      <div class="mt-4 flex flex-wrap gap-2">
        {#each character.reminders as reminder}
          <ReminderToken
            characterId={character.id}
            characterName={character.name}
            text={reminder}
            edition={character.edition}
            team={character.team}
          />
        {/each}
        {#each character.remindersGlobal as reminder}
          <ReminderToken
            characterId={character.id}
            characterName={character.name}
            text={reminder}
            edition={character.edition}
            team={character.team}
          />
        {/each}
      </div>
    {/if}

    <!-- Links + Start Game -->
    <div class="mt-5 flex items-center gap-4 border-t border-border pt-4">
      <a
        href="/almanac/{character.id}?from={encodeURIComponent(
          page.url.pathname + page.url.search,
        )}"
        class="flex items-center gap-1.5 text-sm font-medium text-indigo-500 transition-colors hover:text-indigo-400"
      >
        <svg
          class="h-4 w-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25"
          />
        </svg>
        Almanac
      </a>
      <a
        href="https://wiki.bloodontheclocktower.com/{encodeURIComponent(
          character.name.replace(/ /g, '_'),
        )}"
        target="_blank"
        rel="noopener noreferrer"
        class="flex items-center gap-1.5 text-sm font-medium text-secondary transition-colors hover:text-primary"
      >
        <svg
          class="h-4 w-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M13.5 6H5.25A2.25 2.25 0 003 8.25v10.5A2.25 2.25 0 005.25 21h10.5A2.25 2.25 0 0018 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25"
          />
        </svg>
        Wiki
      </a>
      <div class="flex-1"></div>
      {#if onstartgame && canStartGame}
        <button
          onclick={() => {
            onclose();
            onstartgame?.();
          }}
          class="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-green-500"
        >
          Start Game
        </button>
      {:else if scriptId}
        <a
          href="/games?script={scriptId}"
          class="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-green-500"
        >
          Start Game
        </a>
      {/if}
    </div>
  </div>
</div>
