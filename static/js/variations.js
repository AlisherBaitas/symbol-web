"use strict";

// Template integration: load <script src="/static/js/variations.js" defer></script>.
// POST /api/variations accepts {text} and returns {variations: [{text,
// description, suggested_banner}]}. The HTTP handler belongs to the handlers team.
(() => {
  const MAX_TEXT_LENGTH = 1000;
  const MAX_VARIATION_LENGTH = 49;
  const MAX_DESCRIPTION_LENGTH = 29;
  // The backend has 10 seconds for the LLM call; allow time for the HTTP round trip.
  const REQUEST_TIMEOUT_MS = 12000;
  const BANNERS = new Set(["standard", "shadow", "thinkertoy"]);
  const INVALID_RESPONSE = "The server returned invalid ideas. Please try again.";

  function validText(text) {
    return typeof text === "string" && text.trim().length > 0 &&
      text.length <= MAX_TEXT_LENGTH && !/[^\x20-\x7E\r\n]/.test(text) &&
      !/\r(?!\n)/.test(text);
  }

  function initializeVariations() {
    const button = document.querySelector("#get-variations-btn");
    const panel = document.querySelector("#variations-modal");
    const input = document.querySelector('[name="inputWord"]');
    const bannerSelect = document.querySelector("#banners");

    // Other pages, including studio.html, manage their own controls in studio.js.
    if (!button || !panel || !input || !bannerSelect) return;
    if (button.dataset.variationsInitialized === "true") return;
    button.dataset.variationsInitialized = "true";
    button.type = "button";
    button.setAttribute("aria-controls", panel.id);
    button.setAttribute("aria-expanded", "false");

    const title = document.createElement("h2");
    title.id = "variations-title";
    title.textContent = "Creative ideas";

    const closeButton = document.createElement("button");
    closeButton.type = "button";
    closeButton.textContent = "Close ideas";

    const status = document.createElement("p");
    status.setAttribute("role", "status");
    status.setAttribute("aria-live", "polite");

    const list = document.createElement("ul");
    panel.replaceChildren(title, closeButton, status, list);
    panel.setAttribute("role", "region");
    panel.setAttribute("aria-labelledby", title.id);
    panel.hidden = true;

    let activeRequest = null;

    function setLoading(loading) {
      button.disabled = loading;
      button.setAttribute("aria-busy", String(loading));
      panel.setAttribute("aria-busy", String(loading));
    }

    function cancelRequest() {
      const request = activeRequest;
      activeRequest = null;
      if (request) {
        clearTimeout(request.timeout);
        request.controller.abort();
      }
      setLoading(false);
    }

    function closeIdeas() {
      cancelRequest();
      panel.hidden = true;
      panel.classList.add("hidden");
      button.setAttribute("aria-expanded", "false");
      list.replaceChildren();
      status.textContent = "";
    }

    function showMessage(message) {
      panel.hidden = false;
      panel.classList.remove("hidden");
      button.setAttribute("aria-expanded", "true");
      list.replaceChildren();
      status.textContent = message;
    }

    function readVariations(data) {
      if (!data || !Array.isArray(data.variations) ||
          data.variations.length < 3 || data.variations.length > 5) {
        throw new Error(INVALID_RESPONSE);
      }
      const seenTexts = new Set();
      for (const variation of data.variations) {
        if (!variation || !validText(variation.text) ||
            variation.text.length > MAX_VARIATION_LENGTH || /[\r\n]/.test(variation.text) ||
            typeof variation.description !== "string" ||
            !variation.description.trim() ||
            Array.from(variation.description).length > MAX_DESCRIPTION_LENGTH ||
            /[\x00-\x1F\x7F-\x9F]/.test(variation.description) ||
            !BANNERS.has(variation.suggested_banner) || seenTexts.has(variation.text)) {
          throw new Error(INVALID_RESPONSE);
        }
        seenTexts.add(variation.text);
      }
      return data.variations;
    }

    function selectVariation(variation) {
      // The original form uses standard.txt; other select menus may use standard.
      const option = Array.from(bannerSelect.options).find((item) =>
        item.value === variation.suggested_banner ||
        item.value === `${variation.suggested_banner}.txt`);
      if (!option) {
        showMessage("The suggested banner is unavailable in this form.");
        return;
      }

      closeIdeas();
      input.value = variation.text;
      bannerSelect.value = option.value;
      // Notify suggestions, character counters, and any other form listeners.
      input.dispatchEvent(new Event("input", { bubbles: true }));
      bannerSelect.dispatchEvent(new Event("change", { bubbles: true }));
      input.focus();
    }

    function renderVariations(variations) {
      const items = variations.map((variation) => {
        const item = document.createElement("li");
        const choice = document.createElement("button");
        choice.type = "button";
        choice.className = "variation";
        const text = document.createElement("strong");
        text.textContent = variation.text;
        const description = document.createElement("span");
        description.textContent = ` — ${variation.description} (${variation.suggested_banner})`;
        choice.append(text, description);
        choice.addEventListener("click", () => selectVariation(variation));
        item.append(choice);
        return item;
      });
      list.replaceChildren(...items);
      status.textContent = "Choose an idea to use its text and banner.";
    }

    async function getVariations() {
      cancelRequest();
      const text = input.value;
      if (!validText(text)) {
        showMessage("Enter 1–1000 characters: English letters, numbers, punctuation, or line breaks.");
        input.focus();
        return;
      }

      const request = { text, controller: new AbortController(), timedOut: false };
      activeRequest = request;
      request.timeout = setTimeout(() => {
        request.timedOut = true;
        request.controller.abort();
      }, REQUEST_TIMEOUT_MS);
      showMessage("Creating ideas…");
      setLoading(true);

      // Identity protects against an older response, even after editing A → B → A.
      function isCurrentRequest() {
        return activeRequest === request && input.value === text;
      }

      try {
        const response = await fetch("/api/variations", {
          method: "POST",
          headers: { "Content-Type": "application/json", Accept: "application/json" },
          body: JSON.stringify({ text }),
          signal: request.controller.signal,
        });
        if (!isCurrentRequest()) return;
        if (request.timedOut) throw new Error("Request timed out");
        if (!response.ok) {
          const message = response.status === 400 ? "The text was rejected. Check your input and try again." :
            response.status === 503 ? "The ideas service is unavailable. Please try again later." :
            "Could not create ideas. Please try again.";
          throw new Error(message);
        }
        let data;
        try {
          data = await response.json();
        } catch {
          throw new Error(INVALID_RESPONSE);
        }
        if (!isCurrentRequest()) return;
        if (request.timedOut) throw new Error("Request timed out");
        renderVariations(readVariations(data));
      } catch (error) {
        if (!isCurrentRequest()) return;
        if (request.timedOut) {
          showMessage("The server took too long to respond. Please try again.");
        } else if (!request.controller.signal.aborted) {
          showMessage(error instanceof TypeError ?
            "Could not connect to the server. Check your connection and try again." : error.message);
        }
      } finally {
        clearTimeout(request.timeout);
        if (activeRequest === request) {
          activeRequest = null;
          setLoading(false);
        }
      }
    }

    button.addEventListener("click", getVariations);
    closeButton.addEventListener("click", () => { closeIdeas(); button.focus(); });
    input.addEventListener("input", closeIdeas);
    input.addEventListener("change", closeIdeas);
    document.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && !panel.hidden) {
        closeIdeas();
        button.focus();
      }
    });
    if (input.form) input.form.addEventListener("submit", closeIdeas);
    window.addEventListener("pagehide", cancelRequest);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initializeVariations, { once: true });
  } else {
    initializeVariations();
  }
})();
