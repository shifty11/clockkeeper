<script lang="ts">
	import { page } from '$app/state';
	import type { Character, Game } from '~/lib/gen/clockkeeper/v1/clockkeeper_pb';
	import { Team } from '~/lib/gen/clockkeeper/v1/clockkeeper_pb';

	let {
		game,
		scriptCharacters = [],
	}: {
		game: Game;
		scriptCharacters?: Character[];
	} = $props();

	interface NightEntry {
		id: string;
		name: string;
		reminder: string;
		team?: number;
		edition?: string;
		isSpecial: boolean;
		inPlay: boolean;
	}

	const SPECIAL_ENTRIES: Record<string, { name: string; reminder: string; position: { first: number; other: number }; minPlayers?: number }> = {
		dusk: { name: 'Dusk', reminder: 'Night begins. All players close their eyes.', position: { first: 0, other: 0 } },
		minioninfo: { name: 'Minion Info', reminder: 'Show the *THIS IS THE DEMON* token. Point to the Demon. Show the *THESE ARE YOUR MINIONS* token. Point to the other Minions.', position: { first: 20, other: -1 }, minPlayers: 7 },
		demoninfo: { name: 'Demon Info', reminder: 'Show the *THESE ARE YOUR MINIONS* token. Point to all Minions. Show the *THESE CHARACTERS ARE NOT IN PLAY* token. Show 3 not-in-play good character tokens.', position: { first: 25, other: -1 }, minPlayers: 7 },
		dawn: { name: 'Dawn', reminder: 'Night ends. All players open their eyes.', position: { first: 999, other: 999 } },
	};

	const teamColors: Record<number, string> = {
		[Team.TOWNSFOLK]: 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950/40',
		[Team.OUTSIDER]: 'border-cyan-200 bg-cyan-50 dark:border-cyan-800 dark:bg-cyan-950/40',
		[Team.MINION]: 'border-orange-200 bg-orange-50 dark:border-orange-800 dark:bg-orange-950/40',
		[Team.DEMON]: 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/40',
		[Team.TRAVELLER]: 'card-traveller',
		[Team.FABLED]: 'border-yellow-300 bg-yellow-50 dark:border-yellow-700 dark:bg-yellow-950/40',
		[Team.LORIC]: 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950/40',
	};

	const unselectedColors: Record<number, string> = {
		[Team.TOWNSFOLK]: 'border-blue-100 bg-blue-50/50 dark:border-blue-800/50 dark:bg-blue-950/20',
		[Team.OUTSIDER]: 'border-cyan-100 bg-cyan-50/50 dark:border-cyan-800/50 dark:bg-cyan-950/20',
		[Team.MINION]: 'border-orange-100 bg-orange-50/50 dark:border-orange-800/50 dark:bg-orange-950/20',
		[Team.DEMON]: 'border-red-100 bg-red-50/50 dark:border-red-800/50 dark:bg-red-950/20',
		[Team.TRAVELLER]: 'border-purple-200 bg-purple-50/30 dark:border-purple-800/50 dark:bg-purple-950/20',
		[Team.FABLED]: 'border-yellow-200 bg-yellow-50/50 dark:border-yellow-700/50 dark:bg-yellow-950/20',
		[Team.LORIC]: 'border-green-100 bg-green-50/50 dark:border-green-800/50 dark:bg-green-950/20',
	};

	const teamNameColors: Record<number, string> = {
		[Team.TOWNSFOLK]: 'text-blue-700 dark:text-blue-300',
		[Team.OUTSIDER]: 'text-cyan-700 dark:text-cyan-300',
		[Team.MINION]: 'text-orange-700 dark:text-orange-300',
		[Team.DEMON]: 'text-red-700 dark:text-red-300',
		[Team.TRAVELLER]: 'text-purple-700 dark:text-purple-300',
		[Team.FABLED]: 'text-yellow-700 dark:text-yellow-300',
		[Team.LORIC]: 'text-green-700 dark:text-green-300',
	};

	const teamDataAttr: Record<number, string> = {
		[Team.TOWNSFOLK]: 'townsfolk',
		[Team.OUTSIDER]: 'outsider',
		[Team.MINION]: 'minion',
		[Team.DEMON]: 'demon',
		[Team.TRAVELLER]: 'traveller',
		[Team.FABLED]: 'fabled',
		[Team.LORIC]: 'loric',
	};

	function iconSuffix(team: number): string {
		if (team === Team.TOWNSFOLK || team === Team.OUTSIDER) return '_g';
		if (team === Team.MINION || team === Team.DEMON) return '_e';
		return '';
	}

	// All characters in play (selected roles + travellers + extras).
	const allSelectedChars = $derived([
		...(game.selectedCharacters ?? []),
		...(game.selectedTravellerCharacters ?? []),
		...(game.extraCharacterDetails ?? []),
	]);

	const selectedIdSet = $derived(new Set(allSelectedChars.map((c) => c.id)));

	// All characters from the script (+ travellers + extras), deduped.
	const allScriptChars = $derived.by(() => {
		const seen = new Set<string>();
		const result: Character[] = [];
		for (const c of [...scriptCharacters, ...(game.selectedTravellerCharacters ?? []), ...(game.extraCharacterDetails ?? [])]) {
			if (!seen.has(c.id)) {
				seen.add(c.id);
				result.push(c);
			}
		}
		return result;
	});

	let showAll = $state(false);

	function buildNightOrder(night: 'first' | 'other'): NightEntry[] {
		const posField = night === 'first' ? 'firstNight' : 'otherNight';
		const reminderField = night === 'first' ? 'firstNightReminder' : 'otherNightReminder';

		const source = showAll ? allScriptChars : allSelectedChars;

		// Character entries — filter by having a reminder, sort by position (or end if no position).
		const charEntries: (NightEntry & { pos: number })[] = source
			.filter((c) => c[reminderField])
			.map((c) => ({
				id: c.id,
				name: c.name,
				reminder: c[reminderField],
				team: c.team,
				edition: c.edition,
				isSpecial: false,
				inPlay: selectedIdSet.has(c.id),
				pos: c[posField] || 500,
			}));

		// Special entries.
		const specialEntries: (NightEntry & { pos: number })[] = [];
		for (const [id, entry] of Object.entries(SPECIAL_ENTRIES)) {
			const pos = night === 'first' ? entry.position.first : entry.position.other;
			if (pos < 0) continue;
			if (entry.minPlayers && game.playerCount < entry.minPlayers) continue;
			specialEntries.push({
				id,
				name: entry.name,
				reminder: entry.reminder,
				isSpecial: true,
				inPlay: true,
				pos,
			});
		}

		const all = [...charEntries, ...specialEntries];
		all.sort((a, b) => a.pos - b.pos);
		return all;
	}

	const firstNightOrder = $derived(buildNightOrder('first'));
	const otherNightOrder = $derived(buildNightOrder('other'));

	let activeNight = $state<'first' | 'other'>('first');
	const activeOrder = $derived(activeNight === 'first' ? firstNightOrder : otherNightOrder);

	/** Replace *TEXT* with bold and :reminder: with a pin icon for {@html} rendering. */
	function formatReminder(text: string): string {
		const escaped = text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
		return escaped
			.replace(/\*([^*]+)\*/g, '<strong class="font-semibold text-primary">$1</strong>')
			.replace(/:reminder:/g, '<span class="inline-flex align-text-bottom" title="Place reminder token"><svg class="h-4.5 w-4.5 text-amber-500" fill="currentColor" viewBox="0 0 20 20"><path d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" /></svg></span>');
	}

	const specialIcons: Record<string, string> = {
		dusk: '/night-dusk.webp',
		dawn: '/night-dawn.webp',
		minioninfo: '/night-minioninfo.webp',
		demoninfo: '/night-demoninfo.webp',
	};

	function handlePrint() {
		window.print();
	}
</script>

<div class="space-y-4">
	<!-- Print-only title -->
	<h2 class="print-title hidden text-xl font-bold">{activeNight === 'first' ? 'First Night' : 'Other Nights'}</h2>

	<!-- Header with sub-tabs, toggle, and print button -->
	<div class="no-print flex items-center justify-between">
		<div class="flex items-center gap-3">
			<div class="flex gap-1 rounded-lg bg-element p-1">
				<button
					onclick={() => (activeNight = 'first')}
					class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors {activeNight === 'first'
						? 'bg-surface text-primary shadow-sm'
						: 'text-secondary hover:text-medium'}"
				>
					First Night
				</button>
				<button
					onclick={() => (activeNight = 'other')}
					class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors {activeNight === 'other'
						? 'bg-surface text-primary shadow-sm'
						: 'text-secondary hover:text-medium'}"
				>
					Other Nights
				</button>
			</div>
			<button
				onclick={() => (showAll = !showAll)}
				class="flex items-center gap-1.5 text-xs text-secondary"
			>
				<div
					class="flex h-5 w-5 shrink-0 items-center justify-center rounded border transition-colors {showAll
						? 'border-green-500 bg-green-500'
						: 'border-border-strong'}"
				>
					{#if showAll}
						<svg class="h-3 w-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
						</svg>
					{/if}
				</div>
				Show all
			</button>
		</div>
		<button
			onclick={handlePrint}
			class="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-sm text-secondary transition-colors hover:bg-hover hover:text-medium"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
			</svg>
			Print
		</button>
	</div>

	<!-- Night order list (screen) -->
	<div class="space-y-1">
		{#if activeOrder.length === 0}
			<p class="py-8 text-center text-sm text-muted">No characters with night actions selected.</p>
		{:else}
			{#each activeOrder as entry, i (entry.id)}
				{#if entry.isSpecial}
					<div class="flex items-center gap-3 rounded-lg bg-element/50 px-3 py-2.5">
						<img
							src={specialIcons[entry.id]}
							alt=""
							class="h-20 w-20 shrink-0 object-contain"
							onerror={(e: Event) => ((e.target as HTMLImageElement).style.display = 'none')}
						/>
						<div class="min-w-0 flex-1">
							<span class="text-base font-bold text-primary">{entry.name}</span>
							<p class="text-sm text-muted">{@html formatReminder(entry.reminder)}</p>
						</div>
						<span class="w-6 shrink-0 text-center text-xs font-bold text-muted">{i + 1}</span>
					</div>
				{:else}
					<div
						class="card-slate flex items-center gap-3 rounded-lg border px-3 py-2.5 {entry.inPlay
							? (teamColors[entry.team ?? 0] ?? 'border-border')
							: (unselectedColors[entry.team ?? 0] ?? 'border-border/50') + ' opacity-40 border-dashed'}"
						data-team={teamDataAttr[entry.team ?? 0] ?? ''}
					>
						<img
							src="/characters/{entry.edition}/{entry.id}{iconSuffix(entry.team ?? 0)}.webp"
							alt=""
							class="h-20 w-20 shrink-0 rounded-full"
							onerror={(e: Event) => ((e.target as HTMLImageElement).style.display = 'none')}
						/>
						<div class="min-w-0 flex-1">
							<span class="text-base font-medium {teamNameColors[entry.team ?? 0] ?? 'text-primary'}">{entry.name}</span>
							<p class="text-sm text-secondary">{@html formatReminder(entry.reminder)}</p>
						</div>
						<div class="no-print flex shrink-0 items-center gap-1">
							<a href="/almanac/{entry.id}?from={encodeURIComponent(page.url.pathname + page.url.search)}" class="rounded p-1 text-muted transition-colors hover:bg-hover hover:text-medium" title="Almanac">
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" />
								</svg>
							</a>
							<a href="https://wiki.bloodontheclocktower.com/{entry.name.replace(/ /g, '_')}" target="_blank" rel="noopener" class="rounded p-1 text-muted transition-colors hover:bg-hover hover:text-medium" title="Wiki">
								<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
								</svg>
							</a>
						</div>
						<span class="w-6 shrink-0 text-center text-xs font-medium text-muted">{i + 1}</span>
					</div>
				{/if}
			{/each}
		{/if}
	</div>
</div>
