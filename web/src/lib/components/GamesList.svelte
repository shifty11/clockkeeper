<script lang="ts">
  import { client } from "~/lib/api";
  import { invalidateSidebar } from "~/lib/sidebar-data.svelte";
  import {
    GameState,
    PhaseType,
  } from "~/lib/gen/clockkeeper/v1/clockkeeper_pb";
  import type { GameSummary } from "~/lib/gen/clockkeeper/v1/clockkeeper_pb";

  let {
    games,
    pageSize = 5,
    ondeleted,
  }: {
    games: GameSummary[];
    pageSize?: number;
    ondeleted: (id: bigint) => void;
  } = $props();

  let currentPage = $state(0);
  let confirmingDelete = $state<bigint | null>(null);
  let deleting = $state(false);

  async function deleteGame(id: bigint) {
    deleting = true;
    try {
      await client.deleteGame({ id });
      confirmingDelete = null;
      invalidateSidebar();
      ondeleted(id);
    } catch {
      // Silently fail — user can retry
    } finally {
      deleting = false;
    }
  }

  const totalPages = $derived(Math.ceil(games.length / pageSize));
  const pagedGames = $derived(
    games.slice(currentPage * pageSize, (currentPage + 1) * pageSize),
  );

  $effect(() => {
    if (currentPage >= totalPages && totalPages > 0) {
      currentPage = totalPages - 1;
    }
  });

  function stateBadge(game: GameSummary): { label: string; classes: string } {
    switch (game.state) {
      case GameState.SETUP:
        return { label: "Setup", classes: "bg-yellow-100 text-yellow-700" };
      case GameState.IN_PROGRESS: {
        const phase = game.currentPhaseType === PhaseType.DAY ? "Day" : "Night";
        return {
          label: `${phase} ${game.currentRound}`,
          classes: "bg-green-100 text-green-700",
        };
      }
      case GameState.COMPLETED:
        return { label: "Completed", classes: "bg-element text-muted" };
      default:
        return { label: "Unknown", classes: "bg-element text-muted" };
    }
  }
</script>

<div class="grid gap-3">
  {#each pagedGames as game (game.id)}
    {@const badge = stateBadge(game)}
    {#if confirmingDelete === game.id}
      <div
        class="card-slate rounded-xl border border-red-300 bg-surface p-4 dark:border-red-800"
      >
        <div class="flex items-center justify-between">
          <span class="text-sm text-primary"
            >Delete <strong>{game.scriptName}</strong>?</span
          >
          <div class="flex gap-2">
            <button
              onclick={() => (confirmingDelete = null)}
              class="rounded-lg px-3 py-1 text-sm text-secondary transition-colors hover:bg-hover hover:text-primary"
            >
              Cancel
            </button>
            <button
              onclick={() => deleteGame(game.id)}
              disabled={deleting}
              class="rounded-lg bg-red-600 px-3 py-1 text-sm text-white transition-colors hover:bg-red-700 disabled:opacity-50"
            >
              {deleting ? "Deleting..." : "Delete"}
            </button>
          </div>
        </div>
      </div>
    {:else}
      <div
        class="card-slate group relative rounded-xl border border-border bg-surface transition-all hover:border-indigo-400 hover:shadow-md"
      >
        <a href="/games/{game.id}" class="block p-4">
          <div class="flex items-center justify-between pr-8">
            <span class="font-medium text-primary"
              >{game.name || game.scriptName}</span
            >
            <span
              class="rounded-full px-2.5 py-0.5 text-xs font-medium {badge.classes}"
              >{badge.label}</span
            >
          </div>
          {#if game.name && game.scriptName}
            <p class="mt-0.5 text-xs text-muted">{game.scriptName}</p>
          {/if}
          <div class="mt-1.5 flex gap-3 text-sm text-secondary">
            <span>{game.playerCount} players</span>
            {#if game.deathCount > 0}
              <span>&middot;</span>
              <span
                >{game.deathCount}
                {game.deathCount === 1 ? "death" : "deaths"}</span
              >
            {/if}
          </div>
        </a>
        <button
          onclick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            confirmingDelete = game.id;
          }}
          class="absolute top-3 right-3 rounded-lg p-1.5 text-muted opacity-0 transition-all hover:bg-hover hover:text-red-500 group-hover:opacity-100"
          aria-label="Delete game"
        >
          <svg
            class="h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0"
            />
          </svg>
        </button>
      </div>
    {/if}
  {/each}
</div>

{#if totalPages > 1}
  <div class="mt-4 flex items-center justify-center gap-3">
    <button
      onclick={() => currentPage--}
      disabled={currentPage === 0}
      class="rounded-lg px-3 py-1.5 text-sm text-secondary transition-colors hover:bg-hover hover:text-primary disabled:opacity-40 disabled:hover:bg-transparent"
    >
      Previous
    </button>
    <span class="text-sm text-muted">{currentPage + 1} / {totalPages}</span>
    <button
      onclick={() => currentPage++}
      disabled={currentPage >= totalPages - 1}
      class="rounded-lg px-3 py-1.5 text-sm text-secondary transition-colors hover:bg-hover hover:text-primary disabled:opacity-40 disabled:hover:bg-transparent"
    >
      Next
    </button>
  </div>
{/if}
