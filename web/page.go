package web

const indexHTML = `<!doctype html>
<html lang="de">
<head>
	<meta charset="utf-8">
	<meta
		name="viewport"
		content="width=device-width, initial-scale=1"
	>

	<title>ravendex Enemy Tracker</title>

	<style>
		:root {
			color-scheme: dark;
			font-family: Inter, system-ui, sans-serif;
			background: #11131a;
			color: #f4f4f5;
		}

		* {
			box-sizing: border-box;
		}

		body {
			max-width: 900px;
			margin: 0 auto;
			padding: 32px 20px 60px;
		}

		h1,
		h2,
		h3,
		p {
			margin-top: 0;
		}

		h1 {
			margin-bottom: 8px;
		}

		.zone-name-missing {
			color: #f87171;
		}

		button,
		input {
			font: inherit;
		}

		.subtitle {
			margin-bottom: 24px;
			color: #a1a1aa;
		}

		.card {
			margin-top: 20px;
			padding: 24px;
			border: 1px solid #27272a;
			border-radius: 16px;
			background: #18181b;
		}









		.enemy-list {
			display: grid;
			gap: 18px;
			margin-top: 20px;
		}

		.enemy-card {
			padding: 20px;
			border: 1px solid #303038;
			border-radius: 14px;
			background: #14161c;
		}

		.enemy-header {
			display: flex;
			align-items: flex-start;
			justify-content: space-between;
			gap: 16px;
			margin-bottom: 16px;
		}

		.enemy-name {
			margin-bottom: 4px;
			font-size: 19px;
		}

		.enemy-name.missing {
			color: #f87171;
		}

		.enemy-name-edit-row {
			display: flex;
			align-items: center;
			gap: 10px;
			margin-bottom: 4px;
		}

		.enemy-name-input {
			width: min(100%, 520px);
			padding: 0;
			border: 0;
			border-bottom: 1px solid transparent;
			border-radius: 0;
			background: transparent;
			color: #f87171;
			font-size: 19px;
			font-weight: 700;
			line-height: 1.2;
			outline: none;
		}

		.enemy-name-input::placeholder {
			color: #f87171;
			opacity: 0.72;
		}

		.enemy-name-input:focus {
			border-bottom-color: #f87171;
		}

		.enemy-name-save {
			flex: 0 0 auto;
			padding: 6px 10px;
			background: #7f1d1d;
			color: #fecaca;
			font-size: 13px;
		}

		.enemy-name-save:hover {
			background: #991b1b;
		}

		.enemy-name-hint,
		.enemy-name-error {
			margin-top: 7px;
			color: #f87171;
			font-size: 12px;
		}

		.enemy-id {
			color: #71717a;
			font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
			font-size: 12px;
			overflow-wrap: anywhere;
		}

		.enemy-count {
			flex: 0 0 auto;
			padding: 6px 10px;
			border-radius: 999px;
			background: #27272a;
			font-weight: 700;
		}

		.drop-heading {
			margin-bottom: 10px;
			color: #d4d4d8;
			font-size: 14px;
		}

		.item-grid {
			display: grid;
			grid-template-columns:
				repeat(auto-fill, minmax(220px, 1fr));
			gap: 10px;
		}

		.item-option {
			display: grid;
			grid-template-columns: auto 1fr 58px;
			align-items: center;
			gap: 10px;
			min-height: 48px;
			padding: 10px 12px;
			border: 1px solid #303038;
			border-radius: 10px;
			background: #1b1d24;
			cursor: pointer;
			transition:
				border-color 120ms ease,
				background 120ms ease;
		}

		.item-option:hover {
			border-color: #52525b;
		}

		.item-option.selected {
			border-color: #7c3aed;
			background: #2e1065;
		}

		.item-option input[type="checkbox"] {
			width: 17px;
			height: 17px;
			accent-color: #8b5cf6;
		}

		.item-name {
			overflow-wrap: anywhere;
		}

		.quantity {
			width: 52px;
			padding: 6px;
			border: 1px solid #3f3f46;
			border-radius: 7px;
			background: #11131a;
			color: #f4f4f5;
			text-align: center;
		}

		.quantity:disabled {
			opacity: 0.4;
		}

		.empty {
			padding: 20px;
			border: 1px dashed #3f3f46;
			border-radius: 12px;
			color: #a1a1aa;
			text-align: center;
		}

		.notice {
			margin-top: 12px;
			color: #a1a1aa;
			font-size: 13px;
		}

				.zone-warning {
			display: flex;
			align-items: flex-start;
			gap: 12px;
			margin-top: 16px;
			padding: 14px 16px;
			border: 1px solid #991b1b;
			border-radius: 10px;
			background: #450a0a;
			color: #fecaca;
		}





		.actions {
			display: flex;
			flex-wrap: wrap;
			gap: 10px;
			margin-top: 20px;
		}

		button {
			padding: 10px 16px;
			border: 0;
			border-radius: 9px;
			cursor: pointer;
			font-weight: 600;
		}

		.primary-button {
			background: #7c3aed;
			color: white;
		}

		.primary-button:hover {
			background: #6d28d9;
		}



		button:disabled {
			cursor: not-allowed;
			opacity: 0.45;
		}

		.report-preview {
			margin-top: 20px;
			padding: 16px;
			border: 1px solid #27272a;
			border-radius: 10px;
			background: #0d0f14;
			overflow-x: auto;
			font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
			font-size: 12px;
			white-space: pre-wrap;
		}
		.zone-sub {
			display: block;
			margin-top: 4px;
			margin-bottom: 24px;
			color: #a1a1aa;
			font-size: 15px;
			font-weight: 400;
		}

		.zone-sub.missing {
			color: #f87171;
		}

		.hidden {
			display: none;
		}

		@media (max-width: 600px) {
			body {
				padding: 20px 14px 40px;
			}

			.card {
				padding: 18px;
			}

			.item-grid {
				grid-template-columns: 1fr;
			}
		}
	</style>
</head>

<body>
	<h1 id="zone-name">—</h1>
	<p id="zone-sub" class="zone-sub hidden"></p>


	<section class="card">
		<h2>Besiegte Gegner und Drops</h2>

		<p class="notice">
			Wähle alle Items aus, die in diesem Kampf gedroppt sind.
			Auch ein bestätigter Kampf ohne Drops muss später gespeichert
			werden, damit die Statistik korrekt bleibt.
		</p>

		<div id="enemy-list" class="enemy-list">
			<div class="empty">
				Noch keine Gegner erfasst.
			</div>
		</div>

		<div class="actions">
			<button
				id="preview-button"
				class="primary-button"
				type="button"
				disabled
			>
				Report prüfen
			</button>
		</div>

		<pre
			id="report-preview"
			class="report-preview hidden"
		></pre>
	</section>

	<script>

		const zoneNameElement =
			document.getElementById("zone-name");

		const zoneSubElement =
			document.getElementById("zone-sub");

		const enemyListElement =
			document.getElementById("enemy-list");

		const previewButton =
			document.getElementById("preview-button");

		const previewElement =
			document.getElementById("report-preview");

		let currentState = null;
		let currentDuelID = "";

		/*
			Map-Aufbau:

			enemyId -> Map(
				itemId -> quantity
			)
		*/
		const selectedDrops = new Map();
		const enemyNameDrafts = new Map();
		const savedEnemyNames = new Map();

		function isEditingEnemyName() {
			return document.activeElement?.classList.contains(
				"enemy-name-input"
			);
		}

		function getEnemySelections(enemyId) {
			if (!selectedDrops.has(enemyId)) {
				selectedDrops.set(enemyId, new Map());
			}

			return selectedDrops.get(enemyId);
		}

		async function saveEnemyName(
			enemy,
			input,
			button,
			errorElement
		) {
			const enemyName = input.value.trim();

			if (!enemyName) {
				errorElement.textContent =
					"Bitte einen deutschen Namen eingeben.";
				input.focus();
				return;
			}

			button.disabled = true;
			input.disabled = true;
			errorElement.textContent = "";

			try {
				const response = await fetch("/api/enemy-name", {
					method: "POST",
					headers: {
						"Content-Type": "application/json"
					},
					body: JSON.stringify({
						enemyId: enemy.enemyId,
						enemyName: enemyName
					})
				});

				if (!response.ok) {
					throw new Error(await response.text());
				}

				savedEnemyNames.set(enemy.enemyId, enemyName);
				enemyNameDrafts.delete(enemy.enemyId);

				enemy.enemyName = enemyName;
				enemy.nameMissing = false;

				renderEnemies(currentState?.enemies || []);
				hidePreview();
			} catch (error) {
				console.error(
					"Failed to save enemy name:",
					error
				);

				errorElement.textContent =
					"Der Name konnte nicht gespeichert werden.";
				button.disabled = false;
				input.disabled = false;
				input.focus();
			}
		}

		function renderEnemies(enemies) {
			enemyListElement.replaceChildren();

			if (!Array.isArray(enemies) || enemies.length === 0) {
				const empty = document.createElement("div");
				empty.className = "empty";
				empty.textContent = "Noch keine Gegner erfasst.";

				enemyListElement.appendChild(empty);
				previewButton.disabled = true;
				return;
			}

			previewButton.disabled = false;

			for (const enemy of enemies) {
				const card = document.createElement("article");
				card.className = "enemy-card";

				const header = document.createElement("div");
				header.className = "enemy-header";

				const identity = document.createElement("div");

				const locallySavedName =
					savedEnemyNames.get(enemy.enemyId);

				if (locallySavedName) {
					enemy.enemyName = locallySavedName;
					enemy.nameMissing = false;
				}

				if (enemy.nameMissing) {
					const editRow = document.createElement("div");
					editRow.className = "enemy-name-edit-row";

					const input = document.createElement("input");
					input.type = "text";
					input.className = "enemy-name-input";
					input.placeholder =
						enemy.enemyName || enemy.enemyId;
					input.maxLength = 120;
					input.autocomplete = "off";
					input.spellcheck = false;
					input.value =
						enemyNameDrafts.get(enemy.enemyId) || "";

					const saveButton =
						document.createElement("button");

					saveButton.type = "button";
					saveButton.className = "enemy-name-save";
					saveButton.textContent = "Speichern";

					const errorElement =
						document.createElement("div");

					errorElement.className = "enemy-name-error";

					input.addEventListener("input", () => {
						enemyNameDrafts.set(
							enemy.enemyId,
							input.value
						);
						errorElement.textContent = "";
					});

					input.addEventListener("keydown", event => {
						if (event.key === "Enter") {
							event.preventDefault();
							saveEnemyName(
								enemy,
								input,
								saveButton,
								errorElement
							);
						}
					});

					saveButton.addEventListener("click", () => {
						saveEnemyName(
							enemy,
							input,
							saveButton,
							errorElement
						);
					});

					editRow.append(input, saveButton);

					const hint = document.createElement("div");
					hint.className = "enemy-name-hint";
					hint.textContent =
						"Deutsche Übersetzung fehlt";

					identity.append(editRow, hint, errorElement);
				} else {
					const name = document.createElement("h3");
					name.className = "enemy-name";
					name.textContent =
						enemy.enemyName || enemy.enemyId;

					identity.appendChild(name);
				}

				const technicalName =
					document.createElement("div");

				technicalName.className = "enemy-id";
				technicalName.textContent = enemy.enemyId;

				identity.appendChild(technicalName);

				const count = document.createElement("div");
				count.className = "enemy-count";
				count.textContent = enemy.count + "×";

				header.append(identity, count);
				card.appendChild(header);

				const dropHeading =
					document.createElement("div");

				dropHeading.className = "drop-heading";
				dropHeading.textContent = "Mögliche Drops";

				card.appendChild(dropHeading);

				const items = Array.isArray(enemy.items)
					? enemy.items
					: [];

				if (items.length === 0) {
					const emptyItems =
						document.createElement("div");

					emptyItems.className = "empty";
					emptyItems.textContent =
						"Für diesen Gegner sind noch keine Drops hinterlegt.";

					card.appendChild(emptyItems);
					enemyListElement.appendChild(card);
					continue;
				}

				const selections =
					getEnemySelections(enemy.enemyId);

				const groupedItems = new Map();

				for (const item of items) {
					const category = (item.id || "").slice(0, 2) || "??";
					if (!groupedItems.has(category)) {
						groupedItems.set(category, []);
					}
					groupedItems.get(category).push(item);
				}

				for (const [category, categoryItems] of groupedItems) {
					const heading = document.createElement("h4");
					heading.textContent = category;
					card.appendChild(heading);

					const itemGrid = document.createElement("div");
					itemGrid.className = "item-grid";

					for (const item of categoryItems) {
					const option =
						document.createElement("label");

					option.className = "item-option";

					const checkbox =
						document.createElement("input");

					checkbox.type = "checkbox";
					checkbox.value = item.id;

					const itemName =
						document.createElement("span");

					itemName.className = "item-name";
					itemName.textContent =
						item.name || item.id;

					const quantity =
						document.createElement("input");

					quantity.type = "number";
					quantity.className = "quantity";
					quantity.min = "1";
					quantity.step = "1";

					const savedQuantity =
						selections.get(item.id);

					if (savedQuantity !== undefined) {
						checkbox.checked = true;
						quantity.disabled = false;
						quantity.value = String(savedQuantity);
						option.classList.add("selected");
					} else {
						checkbox.checked = false;
						quantity.disabled = true;
						quantity.value = "1";
					}

					checkbox.addEventListener("change", () => {
						if (checkbox.checked) {
							const value = Math.max(
								1,
								Number.parseInt(
									quantity.value,
									10
								) || 1
							);

							selections.set(item.id, value);
							quantity.disabled = false;
							option.classList.add("selected");
						} else {
							selections.delete(item.id);
							quantity.disabled = true;
							option.classList.remove("selected");
						}

						hidePreview();
					});

					quantity.addEventListener("input", () => {
						const value = Math.max(
							1,
							Number.parseInt(
								quantity.value,
								10
							) || 1
						);

						quantity.value = String(value);

						if (checkbox.checked) {
							selections.set(item.id, value);
						}

						hidePreview();
					});

					quantity.addEventListener(
						"click",
						event => event.stopPropagation()
					);

					option.append(
						checkbox,
						itemName,
						quantity
					);

						itemGrid.appendChild(option);
					}

					card.appendChild(itemGrid);
				}
				enemyListElement.appendChild(card);
			}
		}

		function buildReport() {
			if (!currentState) {
				return null;
			}

			const reports = currentState.enemies.map(enemy => {
				const selections =
					selectedDrops.get(enemy.enemyId);

				const drops = {};

				if (selections) {
					for (const [itemId, quantity]
						of selections.entries()) {
						drops[itemId] = quantity;
					}
				}

				return {
					enemyId: enemy.enemyId,
					enemyName: enemy.enemyName,
					observations: enemy.count,
					drops: drops
				};
			});

			return {
				duelId: currentState.duelId,
				zoneKey: currentState.zoneKey,
				zone: currentState.zone,
				sub: currentState.sub,
				world: currentState.world,
				zoneResolved: currentState.zoneResolved,
				won: currentState.won,
				reports: reports
			};
		}

		function hidePreview() {
			previewElement.classList.add("hidden");
			previewElement.textContent = "";
		}

		function renderState(state) {
			currentState = state;

			if (state.duelId !== currentDuelID) {
				currentDuelID = state.duelId || "";
				selectedDrops.clear();
				hidePreview();
			}



			const hasUnknownZone =
				Boolean(state.zoneKey) &&
				state.zoneResolved === false;

			zoneNameElement.textContent =
				state.zone || state.zoneKey || "—";

			zoneNameElement.classList.toggle(
				"zone-name-missing",
				hasUnknownZone
			);

			if (hasUnknownZone) {
				zoneSubElement.textContent =
					"Dieses Gebiet ist noch nicht in den Übersetzungsdaten hinterlegt.";
				zoneSubElement.className = "zone-sub missing";
			} else if (state.sub) {
				zoneSubElement.textContent = state.sub;
				zoneSubElement.className = "zone-sub";
			} else {
				zoneSubElement.textContent = "";
				zoneSubElement.className = "zone-sub hidden";
			}

			const enemies = Array.isArray(state.enemies)
				? state.enemies
				: [];


			// Während ein Gegnername bearbeitet wird, bleibt der bestehende
			// DOM erhalten. So verliert das Eingabefeld durch das sekündliche
			// Polling weder Fokus noch Cursorposition.
			if (!isEditingEnemyName()) {
				renderEnemies(enemies);
			}
		}

		async function updateTracker() {
			try {
				const response =
					await fetch("/api/enemies", {
						cache: "no-store"
					});

				if (!response.ok) {
					throw new Error(
						"Request failed: " +
						response.status
					);
				}

				const state = await response.json();
				renderState(state);
			} catch (error) {
				console.error(
					"Failed to update enemy tracker:",
					error
				);
			}
		}

		previewButton.addEventListener("click", () => {
			const report = buildReport();

			if (!report) {
				return;
			}

			previewElement.textContent =
				JSON.stringify(report, null, 2);

			previewElement.classList.remove("hidden");
		});

		

		updateTracker();
		setInterval(updateTracker, 1000);
	</script>
</body>
</html>`
