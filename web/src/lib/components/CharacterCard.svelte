<script lang="ts">
	import type { Character } from '~/lib/gen/clockkeeper/v1/clockkeeper_pb';
	import { Team } from '~/lib/gen/clockkeeper/v1/clockkeeper_pb';

	let {
		character,
		selected = true,
		removable = false,
		onremove
	}: {
		character: Character;
		selected?: boolean;
		removable?: boolean;
		onremove?: () => void;
	} = $props();

	const iconSuffix = $derived(
		character.team === Team.TOWNSFOLK || character.team === Team.OUTSIDER ? '_g' :
		character.team === Team.MINION || character.team === Team.DEMON ? '_e' : ''
	);
	const iconUrl = $derived(`/characters/${character.edition}/${character.id}${iconSuffix}.webp`);

	const teamColors: Record<number, string> = {
		[Team.TOWNSFOLK]: 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950/40',
		[Team.OUTSIDER]: 'border-cyan-200 bg-cyan-50 dark:border-cyan-800 dark:bg-cyan-950/40',
		[Team.MINION]: 'border-orange-200 bg-orange-50 dark:border-orange-800 dark:bg-orange-950/40',
		[Team.DEMON]: 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950/40',
		[Team.TRAVELLER]: 'card-traveller',
		[Team.FABLED]: 'border-yellow-300 bg-yellow-50 dark:border-yellow-700 dark:bg-yellow-950/40',
		[Team.LORIC]: 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950/40'
	};

	const unselectedColors: Record<number, string> = {
		[Team.TOWNSFOLK]: 'border-blue-100 bg-blue-50/50 dark:border-blue-800/50 dark:bg-blue-950/20',
		[Team.OUTSIDER]: 'border-cyan-100 bg-cyan-50/50 dark:border-cyan-800/50 dark:bg-cyan-950/20',
		[Team.MINION]: 'border-orange-100 bg-orange-50/50 dark:border-orange-800/50 dark:bg-orange-950/20',
		[Team.DEMON]: 'border-red-100 bg-red-50/50 dark:border-red-800/50 dark:bg-red-950/20',
		[Team.TRAVELLER]: 'border-purple-200 bg-purple-50/30 dark:border-purple-800/50 dark:bg-purple-950/20',
		[Team.FABLED]: 'border-yellow-200 bg-yellow-50/50 dark:border-yellow-700/50 dark:bg-yellow-950/20',
		[Team.LORIC]: 'border-green-100 bg-green-50/50 dark:border-green-800/50 dark:bg-green-950/20'
	};

	const teamBadgeColors: Record<number, string> = {
		[Team.TOWNSFOLK]: 'bg-blue-100 text-blue-700 dark:bg-blue-500/20 dark:text-blue-300',
		[Team.OUTSIDER]: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-500/20 dark:text-cyan-300',
		[Team.MINION]: 'bg-orange-100 text-orange-700 dark:bg-orange-500/20 dark:text-orange-300',
		[Team.DEMON]: 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-300',
		[Team.TRAVELLER]: 'bg-purple-100 text-purple-700 dark:bg-purple-500/20 dark:text-purple-300',
		[Team.FABLED]: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-500/20 dark:text-yellow-300',
		[Team.LORIC]: 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-300'
	};

	const teamNames: Record<number, string> = {
		[Team.TOWNSFOLK]: 'Townsfolk',
		[Team.OUTSIDER]: 'Outsider',
		[Team.MINION]: 'Minion',
		[Team.DEMON]: 'Demon',
		[Team.TRAVELLER]: 'Traveller',
		[Team.FABLED]: 'Fabled',
		[Team.LORIC]: 'Loric'
	};

	const teamDataAttr: Record<number, string> = {
		[Team.TOWNSFOLK]: 'townsfolk',
		[Team.OUTSIDER]: 'outsider',
		[Team.MINION]: 'minion',
		[Team.DEMON]: 'demon',
		[Team.TRAVELLER]: 'traveller',
		[Team.FABLED]: 'fabled',
		[Team.LORIC]: 'loric'
	};

	const colorClass = $derived(
		selected
			? (teamColors[character.team] ?? 'border-border bg-surface-alt')
			: (unselectedColors[character.team] ?? 'border-border/50 bg-surface-alt/30')
	);

	let imgError = $state(false);
</script>

<div class="card-slate h-full rounded-lg border p-2 transition-opacity {colorClass}" class:opacity-40={!selected} class:border-dashed={!selected} data-team={teamDataAttr[character.team] ?? ''}>
	<div class="flex items-center gap-2">
		{#if !imgError}
			<img
				src={iconUrl}
				alt={character.name}
				class="h-24 w-24 shrink-0 rounded-full"
				onerror={() => (imgError = true)}
			/>
		{:else}
			<div class="flex h-24 w-24 shrink-0 items-center justify-center rounded-full bg-element text-sm text-secondary">
				{character.name.charAt(0)}
			</div>
		{/if}
		<div class="min-w-0 flex-1">
			<div class="flex items-center gap-2">
				<span class="font-medium text-primary">{character.name}</span>
				<span class="rounded px-1.5 py-0.5 text-xs {teamBadgeColors[character.team] ?? 'bg-element text-secondary'}">
					{teamNames[character.team] ?? 'Unknown'}
				</span>
				{#if character.setup}
					<span class="rounded bg-yellow-100 px-1.5 py-0.5 text-xs text-yellow-700 dark:bg-yellow-500/20 dark:text-yellow-300">setup</span>
				{/if}
			</div>
			<p class="mt-0.5 text-sm text-secondary">{character.ability}</p>
		</div>
		{#if removable && onremove}
			<button
				onclick={onremove}
				aria-label="Remove {character.name}"
				class="shrink-0 rounded p-1 text-muted transition-colors hover:bg-hover hover:text-label"
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		{/if}
	</div>
</div>
