// Small helper: send/receive JSON. On any 401 we redirect to /login
// immediately, since that means the session is missing or expired;
// callers don't need to handle 401 themselves.
async function api(method, path, body) {
	const res = await fetch(path, {
		method,
		headers: body ? { "Content-Type": "application/json" } : undefined,
		body: body ? JSON.stringify(body) : undefined,
	});

	if (res.status === 401) {
		window.location.href = "/login";
		throw new Error("not logged in");
	}

	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		throw new Error(data.error || "request failed");
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

setupAuthForm("login-form", "/api/login", "/practice");
setupAuthForm("register-form", "/api/register", "/practice");

// --- Shared nav: greeting + logout, present on any page that has it ---

function initNav() {
	const topNav = document.getElementById("top-nav");
	if (!topNav) return; // login/register pages don't render the nav

	const greeting = document.getElementById("greeting");
	const logoutBtn = document.getElementById("logout-btn");

	api("GET", "/api/me")
		.then((me) => showText(greeting, `Hi, ${me.username}!`))
		.catch(() => {}); // api() already redirects to /login on 401

	logoutBtn.addEventListener("click", async () => {
		await api("POST", "/api/logout");
		window.location.href = "/login";
	});
}

// --- Progress page: fetch and display the stat cards ---

function initProgress() {
	const streakEl = document.getElementById("stat-streak");
	if (!streakEl) return; // not on the progress page

	const accuracyEl = document.getElementById("stat-accuracy");
	const totalEl = document.getElementById("stat-total");

	api("GET", "/api/progress")
		.then((p) => {
			showText(streakEl, String(p.streak));
			showText(accuracyEl, Math.round(p.accuracy * 100) + "%");
			showText(totalEl, String(p.total));
		})
		.catch(() => {}); // api() already redirects to /login on 401
}

// --- Practice page: difficulty picker -> 10-question lesson -> score ---

function initPractice() {
	const startScreen = document.getElementById("start-screen");
	if (!startScreen) return; // not on the practice page

	const topNav = document.getElementById("top-nav");
	const lessonScreen = document.getElementById("lesson-screen");
	const completeScreen = document.getElementById("complete-screen");
	const chips = document.querySelectorAll(".chip");
	const backBtn = document.getElementById("back-btn");
	const progressFill = document.getElementById("progress-fill");
	const questionEl = document.getElementById("question");
	const answerInput = document.getElementById("answer-input");
	const lessonError = document.getElementById("lesson-error");
	const checkBtn = document.getElementById("check-btn");
	const feedbackPanel = document.getElementById("feedback-panel");
	const feedbackText = document.getElementById("feedback-text");
	const continueBtn = document.getElementById("continue-btn");
	const scoreText = document.getElementById("score-text");
	const againBtn = document.getElementById("again-btn");

	const LESSON_LENGTH = 10;
	let difficulty = 1;
	let questionIndex = 0;
	let correctCount = 0;
	let currentTaskID = null;
	let answered = false;

	function showScreen(screen) {
		[startScreen, lessonScreen, completeScreen].forEach((s) => s.classList.add("hidden"));
		screen.classList.remove("hidden");
		topNav.classList.toggle("hidden", screen === lessonScreen);
	}

	function updateProgress() {
		progressFill.style.width = (questionIndex / LESSON_LENGTH) * 100 + "%";
	}

	async function loadQuestion() {
		answered = false;
		showText(lessonError, "");
		feedbackPanel.classList.remove("show", "correct", "wrong");
		answerInput.value = "";
		checkBtn.disabled = false;
		updateProgress();
		try {
			const task = await api("GET", `/api/tasks/next?d=${difficulty}`);
			currentTaskID = task.task_id;
			showText(questionEl, task.question);
			answerInput.focus();
		} catch (err) {
			showText(lessonError, err.message);
		}
	}

	function startLesson(difficultyLevel) {
		difficulty = difficultyLevel;
		questionIndex = 0;
		correctCount = 0;
		showScreen(lessonScreen);
		loadQuestion();
	}

	chips.forEach((chip) => {
		chip.addEventListener("click", () => startLesson(Number(chip.dataset.difficulty)));
	});

	backBtn.addEventListener("click", () => showScreen(startScreen));

	async function checkAnswer() {
		if (answered || answerInput.value === "") return;
		showText(lessonError, "");
		try {
			const result = await api("POST", "/api/tasks/answer", {
				task_id: currentTaskID,
				answer: Number(answerInput.value),
			});
			answered = true;
			questionIndex++;
			updateProgress();
			if (result.correct) {
				correctCount++;
				feedbackPanel.classList.add("correct");
				showText(feedbackText, "Nice!");
			} else {
				feedbackPanel.classList.add("wrong");
				showText(feedbackText, `Correct answer: ${result.correct_answer}`);
			}
			feedbackPanel.classList.add("show");
		} catch (err) {
			showText(lessonError, err.message);
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
		if (questionIndex >= LESSON_LENGTH) {
			showText(scoreText, `You got ${correctCount} / ${LESSON_LENGTH} correct.`);
			showScreen(completeScreen);
		} else {
			loadQuestion();
		}
	});

	againBtn.addEventListener("click", () => showScreen(startScreen));

	showScreen(startScreen);
}

initNav();
initPractice();
initProgress();
