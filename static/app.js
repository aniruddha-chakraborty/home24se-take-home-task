const form = document.getElementById("analyze-form");
const feedback = document.getElementById("feedback");
const results = document.getElementById("results");
const submitButton = document.getElementById("submit-button");

const fields = {
  resultURL: document.getElementById("result-url"),
  htmlVersion: document.getElementById("html-version"),
  pageTitle: document.getElementById("page-title"),
  internalLinks: document.getElementById("internal-links"),
  externalLinks: document.getElementById("external-links"),
  brokenLinks: document.getElementById("broken-links"),
  loginForm: document.getElementById("login-form"),
  headingsGrid: document.getElementById("headings-grid"),
};

form.addEventListener("submit", async (event) => {
  event.preventDefault();

  const formData = new FormData(form);
  const url = String(formData.get("url") || "").trim();

  if (!url) {
    showFeedback("Enter a URL first.", "error");
    hideResults();
    return;
  }

  submitButton.disabled = true;
  showFeedback("Analyzing webpage...", "loading");
  hideResults();

  try {
    const response = await fetch("/api/v1/analyze", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ url }),
    });

    const payload = await response.json();

    if (!response.ok || payload.error) {
      const error = payload.error || {
        code: response.status,
        description: "Unknown request failure.",
      };
      showFeedback(`${error.code}: ${error.description}`, "error");
      return;
    }

    renderResult(payload.result);
    showFeedback("Analysis complete.", "success");
  } catch (error) {
    showFeedback(`Request failed: ${error.message}`, "error");
  } finally {
    submitButton.disabled = false;
  }
});

function renderResult(result) {
  fields.resultURL.textContent = result?.analyzedURL || "Unknown URL";
  fields.htmlVersion.textContent = result?.htmlVersion || "Unknown";
  fields.pageTitle.textContent = result?.title || "Not found";
  fields.internalLinks.textContent = String(result?.internalLinks ?? 0);
  fields.externalLinks.textContent = String(result?.externalLinks ?? 0);
  fields.brokenLinks.textContent = String(result?.brokenLinks ?? 0);
  fields.loginForm.textContent = result?.hasLoginForm ? "Detected" : "Not detected";

  fields.headingsGrid.innerHTML = "";
  for (const level of ["h1", "h2", "h3", "h4", "h5", "h6"]) {
    const pill = document.createElement("article");
    pill.className = "heading-pill";
    pill.innerHTML = `<span>${level.toUpperCase()}</span><strong>${result?.headings?.[level] ?? 0}</strong>`;
    fields.headingsGrid.appendChild(pill);
  }

  results.hidden = false;
}

function hideResults() {
  results.hidden = true;
}

function showFeedback(message, state) {
  feedback.hidden = false;
  feedback.textContent = message;
  feedback.className = "feedback";

  if (state === "error") {
    feedback.classList.add("feedback--error");
  } else if (state === "loading") {
    feedback.classList.add("feedback--loading");
  } else if (state === "success") {
    feedback.classList.add("feedback--success");
  }
}
