// Small helper: send/receive JSON. On any 401 we redirect to /login
// immediately, since that means the session is missing or expired;
// callers don't need to handle 401 themselves. The response status is
// attached to thrown errors so callers that DO need to branch on it
// (e.g. the lesson page handling 403/404) can.
async function api(method, path, body) {
	const res = await fetch(path, {
		method,
		headers: body ? { "Content-Type": "application/json" } : undefined,
		body: body ? JSON.stringify(body) : undefined,
	});

	if (res.status === 401) {
		window.location.href = "/login";
		const err = new Error("not logged in");
		err.status = 401;
		throw err;
	}

	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		const err = new Error(data.error || "request failed");
		err.status = res.status;
		throw err;
	}
	return data;
}

// textContent can't be interpreted as HTML/script, so this is safe
// even when the text comes from the server.
function showText(el, text) {
	el.textContent = text;
}

function setupAuthForm(formId, path, redirectTo) {
	const form = document.getElementById(formId);
	if (!form) return;
	const errorEl = document.getElementById("form-error");
	form.addEventListener("submit", async (e) => {
		e.preventDefault();
		showText(errorEl, "");
		const data = Object.fromEntries(new FormData(form));
		try {
			await api("POST", path, data);
			window.location.href = redirectTo;
		} catch (err) {
			showText(errorEl, err.message);
		}
	});
}

setupAuthForm("login-form", "/api/login", "/");
setupAuthForm("register-form", "/api/register", "/");

// --- Levels page: fetch /api/levels and build the path ---

function initLevels() {
	const pathEl = document.getElementById("level-path");
	if (!pathEl) return; // not on the levels page

	api("GET", "/api/levels")
		.then((levels) => {
			pathEl.textContent = "";
			levels.forEach((level) => {
				const node = document.createElement(level.unlocked ? "a" : "div");
				node.className = "level-node " + (level.passed ? "passed" : level.unlocked ? "unlocked" : "locked");
				if (level.unlocked) node.href = `/level/${level.id}`;

				const circle = document.createElement("span");
				circle.className = "level-circle";
				showText(circle, level.passed ? "✓" : level.unlocked ? String(level.id) : "🔒");
				node.appendChild(circle);

				const title = document.createElement("span");
				title.className = "level-title";
				showText(title, level.title);
				node.appendChild(title);

				if (level.passed) {
					const score = document.createElement("span");
					score.className = "level-score";
					showText(score, `${level.best_score}/5`);
					node.appendChild(score);
				}

				pathEl.appendChild(node);
			});
		})
		.catch(() => {}); // api() already redirects to /login on 401
}

// --- Lesson page: start -> 5 questions -> end screen ---

// handleApiError centralizes the lesson page's error handling: 403
// (locked level) and 404 (unknown level/lesson/task) send the user
// home; 401 is already being handled by api() itself. Anything else
// is left for the caller to show inline. Returns true if handled.
function handleApiError(err) {
	if (err.status === 403 || err.status === 404) {
		window.location.href = "/";
		return true;
	}
	return err.status === 401;
}

function initLesson() {
	const root = document.getElementById("lesson-root");
	if (!root) return; // not on the lesson page
	const levelID = root.dataset.levelId;

	const lessonScreen = document.getElementById("lesson-screen");
	const lessonFooter = document.querySelector(".lesson-footer");
	const endScreen = document.getElementById("end-screen");
	const progressFill = document.getElementById("progress-fill");
	const questionEl = document.getElementById("question");
	const answerInput = document.getElementById("answer-input");
	const lessonError = document.getElementById("lesson-error");
	const checkBtn = document.getElementById("check-btn");
	const feedbackPanel = document.getElementById("feedback-panel");
	const feedbackText = document.getElementById("feedback-text");
	const continueBtn = document.getElementById("continue-btn");
	const endTitle = document.getElementById("end-title");
	const endScore = document.getElementById("end-score");
	const endMessage = document.getElementById("end-message");
	const retryBtn = document.getElementById("retry-btn");

	const TOTAL = 5;
	let lessonID = null;
	let currentTaskID = null;
	let currentNumber = 0;
	let lastResult = null;

	function updateProgress(completed) {
		progressFill.style.width = (completed / TOTAL) * 100 + "%";
	}

	async function loadNext() {
		lastResult = null;
		showText(lessonError, "");
		feedbackPanel.classList.remove("show", "correct", "wrong");
		answerInput.value = "";
		try {
			const task = await api("GET", `/api/lessons/${lessonID}/next`);
			currentTaskID = task.task_id;
			currentNumber = task.number;
			updateProgress(currentNumber - 1);
			showText(questionEl, task.question);
			answerInput.focus();
		} catch (err) {
			if (!handleApiError(err)) showText(lessonError, err.message);
		}
	}

	async function startLesson() {
		try {
			const start = await api("POST", `/api/levels/${levelID}/start`);
			lessonID = start.lesson_id;
			await loadNext();
		} catch (err) {
			if (!handleApiError(err)) showText(lessonError, err.message);
		}
	}

	async function checkAnswer() {
		if (lastResult || answerInput.value === "") return;
		showText(lessonError, "");
		try {
			const result = await api("POST", "/api/tasks/answer", {
				task_id: currentTaskID,
				answer: Number(answerInput.value),
			});
			lastResult = result;
			updateProgress(currentNumber);
			if (result.correct) {
				feedbackPanel.classList.add("correct");
				showText(feedbackText, "Nice!");
			} else {
				feedbackPanel.classList.add("wrong");
				showText(feedbackText, `Correct answer: ${result.correct_answer}`);
			}
			feedbackPanel.classList.add("show");
		} catch (err) {
			if (!handleApiError(err)) showText(lessonError, err.message);
		}
	}

	function showEnd() {
		const { score, passed, next_unlocked } = lastResult;

		lessonScreen.classList.add("hidden");
		lessonFooter.classList.add("hidden");
		feedbackPanel.classList.remove("show");
		endScreen.classList.remove("hidden");

		showText(endTitle, passed ? "Level passed!" : "Lesson complete");
		showText(endScore, `${score} / ${TOTAL}`);
		if (passed) {
			showText(endMessage, next_unlocked ? `Level ${Number(levelID) + 1} unlocked.` : "You've mastered this level.");
		} else {
			showText(endMessage, "Need 4/5 to pass. Give it another go!");
		}
	}

	checkBtn.addEventListener("click", checkAnswer);
	answerInput.addEventListener("keydown", (e) => {
		if (e.key === "Enter") {
			e.preventDefault();
			checkAnswer();
		}
	});

	continueBtn.addEventListener("click", () => {
		if (lastResult && lastResult.finished) {
			showEnd();
		} else {
			loadNext();
		}
	});

	retryBtn.addEventListener("click", () => {
		endScreen.classList.add("hidden");
		lessonScreen.classList.remove("hidden");
		lessonFooter.classList.remove("hidden");
		startLesson();
	});

	startLesson();
}

// --- Account page: profile, stats, change password, logout ---

function initAccount() {
	const usernameEl = document.getElementById("account-username");
	if (!usernameEl) return; // not on the account page

	const sinceEl = document.getElementById("account-since");
	const streakEl = document.getElementById("stat-streak");
	const accuracyEl = document.getElementById("stat-accuracy");
	const totalEl = document.getElementById("stat-total");
	const passwordForm = document.getElementById("password-form");
	const passwordStatus = document.getElementById("password-status");
	const logoutBtn = document.getElementById("logout-btn");

	api("GET", "/api/me")
		.then((me) => {
			showText(usernameEl, me.username);
			showText(sinceEl, me.created_at.slice(0, 10));
		})
		.catch(() => {});

	api("GET", "/api/progress")
		.then((p) => {
			showText(streakEl, String(p.streak));
			showText(accuracyEl, Math.round(p.accuracy * 100) + "%");
			showText(totalEl, String(p.total));
		})
		.catch(() => {});

	passwordForm.addEventListener("submit", async (e) => {
		e.preventDefault();
		passwordStatus.classList.remove("success");
		showText(passwordStatus, "");
		const data = Object.fromEntries(new FormData(passwordForm));
		try {
			await api("POST", "/api/account/password", data);
			passwordStatus.classList.add("success");
			showText(passwordStatus, "Password updated.");
			passwordForm.reset();
		} catch (err) {
			showText(passwordStatus, err.message);
		}
	});

	logoutBtn.addEventListener("click", async () => {
		await api("POST", "/api/logout");
		window.location.href = "/login";
	});
}

initLevels();
initLesson();
initAccount();
